package categories

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"kuperparser/internal/apis/kuper"
	"kuperparser/internal/http-server/query"
	"kuperparser/internal/http-server/respond"
)

type Lister interface {
	ListCategories(ctx context.Context, storeID int) ([]kuper.Category, error)
}

type Options struct {
	Log            *slog.Logger
	Lister         Lister
	DefaultStoreID int
	Timeout        time.Duration
	HideRoofLeaf   bool //скрыть категории parent_id=0 && has children = false, тк они не имеют смысла, вывести их нельзя
}

type FlatCategory struct {
	ID            int    `json:"id"`
	ParentID      int    `json:"parent_id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	ProductsCount int    `json:"products_count"`
	HasChildren   bool   `json:"has_children"`
	Level         int    `json:"level"`
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
		if opts.Lister == nil {
			log.Error("categories handler misconfigured: lister is nil")
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
		if storeID <= 0 {
			respond.WriteError(w, 400, "bad_request", "storeID must be > 0")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), opts.Timeout)
		defer cancel()

		tree, err := opts.Lister.ListCategories(ctx, storeID)
		if err != nil {
			log.Error("ListCategories failed", "err", err, "store_id", storeID)
			respond.WriteInternalError(w)
			return
		}

		flat := flatten(tree, 0, opts.HideRoofLeaf)

		respond.WriteJSON(w, 200, map[string]any{
			"store_id":   storeID,
			"fetched_at": time.Now().UTC().Format(time.RFC3339),
			"count":      len(flat),
			"categories": flat,
		})
	}
}
func flatten(cats []kuper.Category, level int, hideRootLeaf bool) []FlatCategory {
	out := make([]FlatCategory, 0, 256)

	for _, c := range cats {
		if hideRootLeaf && c.ParentID == 0 && !c.HasChildren {
			if len(c.Children) > 0 {
				out = append(out, flatten(c.Children, level+1, hideRootLeaf)...)
			}
			continue
		}

		out = append(out, FlatCategory{
			ID:            c.ID,
			ParentID:      c.ParentID,
			Name:          c.Name,
			Slug:          c.Slug,
			ProductsCount: c.ProductsCount,
			HasChildren:   c.HasChildren,
			Level:         level,
		})

		if len(c.Children) > 0 {
			out = append(out, flatten(c.Children, level+1, hideRootLeaf)...)
		}
	}

	return out
}
