package products

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"kuperparser/internal/apis/kuper/endpoints"
	"kuperparser/internal/apis/kuper/responses"
	"kuperparser/internal/domain/models"
	"kuperparser/internal/http-server/query"
	"kuperparser/internal/http-server/respond"
	"kuperparser/internal/repository"
)

type ProductsGetter interface {
	GetByCategoryID(ctx context.Context, storeID int, categoryID int) ([]models.Product, string, error)
}

type StoreGetter interface {
	GetStore(ctx context.Context, storeID int) (responses.StoreInfo, error)
}

type Options struct {
	Log            *slog.Logger
	Products       ProductsGetter
	Store          StoreGetter
	DefaultStoreID int
	Timeout        time.Duration
}

func NewGetHandler(opts Options) http.HandlerFunc {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respond.WriteError(w, 405, "method_not_allowed", "GET only")
			return
		}
		if opts.Products == nil {
			log.Error("products handler misconfigured: ProductsGetter is nil")
			respond.WriteInternalError(w)
			return
		}

		storeID := opts.DefaultStoreID
		if v, present, err := query.IntAny(r, "storeID", "storeid"); err != nil {
			respond.WriteError(w, 400, "bad_request", err.Error())
			return
		} else if present {
			storeID = v
		}

		categoryID, present, err := query.IntAny(r, "categoryID", "categoryid")
		if err != nil {
			respond.WriteError(w, 400, "bad_request", err.Error())
			return
		}
		if !present {
			respond.WriteError(w, 400, "bad_request", "categoryID is required")
			return
		}

		if storeID <= 0 {
			respond.WriteError(w, 400, "bad_request", "storeID must be > 0")
			return
		}
		if categoryID <= 0 {
			respond.WriteError(w, 400, "bad_request", "categoryID must be > 0")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), opts.Timeout)
		defer cancel()

		var storeMeta *repository.StoreMeta
		if opts.Store != nil {
			sm, err := opts.Store.GetStore(ctx, storeID)
			if err != nil {
				log.Warn("GetStore failed (continue)", "err", err, "store_id", storeID)
			} else {
				m := repository.StoreMeta{
					ID:           sm.StoreID,
					Name:         sm.StoreName,
					Address:      sm.StoreAddress,
					RetailerName: sm.RetailerName,
				}
				storeMeta = &m
			}
		}

		products, slug, err := opts.Products.GetByCategoryID(ctx, storeID, categoryID)
		if err != nil {

			var apiErr *endpoints.APIError
			if errors.As(err, &apiErr) {
				if apiErr.Status == http.StatusNotFound {
					respond.WriteError(w, http.StatusNotFound, "not_found", apiErr.Message)
					return
				}
				if apiErr.Status == http.StatusTooManyRequests {
					respond.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
					return
				}
				respond.WriteError(w, http.StatusBadGateway, "upstream_error", apiErr.Error())
				return
			}

			log.Error("GetByCategoryID failed", "err", err, "store_id", storeID, "category_id", categoryID)
			respond.WriteInternalError(w)
			return
		}

		res := repository.CategoryResult{
			FetchedAt: time.Now().UTC().Format(time.RFC3339),
			Store:     storeMeta,
			Category:  &repository.CategoryMeta{ID: categoryID, Slug: slug},
			Products:  products,
			Count:     len(products),
		}

		respond.WriteJSON(w, 200, res)
	}
}
