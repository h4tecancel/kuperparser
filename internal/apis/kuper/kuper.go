package kuper

import (
	"context"
	"log/slog"
	"net/http"

	"kuperparser/internal/apis/kuper/endpoints"
	"kuperparser/internal/apis/kuper/responses"
	"kuperparser/internal/client"
)

type Category = responses.Category
type Product = responses.Product
type StoreInfo = responses.StoreInfo

type KuperService interface {
	ListCategories(ctx context.Context, storeID int) ([]Category, error)
	GetStore(ctx context.Context, storeID int) (StoreInfo, error)
	ListProducts(ctx context.Context, storeID int, departmentSlug string, page, perPage, offersLimit int) ([]Product, error)
}

type service struct {
	api *endpoints.Client
	log *slog.Logger
}

func New(transport client.Transport, baseURL string, logger *slog.Logger) KuperService {
	if baseURL == "" {
		baseURL = "https://kuper.ru"
	}
	if logger == nil {
		logger = slog.Default()
	}

	s := &service{log: logger}
	s.api = endpoints.New(transport, baseURL, s.applyDefaultHeaders)
	return s
}

func (s *service) applyDefaultHeaders(req *http.Request) {
	req.Header.Set(
		"User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
			"AppleWebKit/537.36 (KHTML, like Gecko) "+
			"Chrome/144.0.0.0 Safari/537.36",
	)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.8")
	req.Header.Set("Referer", "https://kuper.ru/")
	req.Header.Set("Origin", "https://kuper.ru")

	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
}

func (s *service) ListCategories(ctx context.Context, storeID int) ([]Category, error) {
	return s.api.ListCategories(ctx, storeID)
}

func (s *service) GetStore(ctx context.Context, storeID int) (StoreInfo, error) {
	return s.api.GetStore(ctx, storeID)
}

func (s *service) ListProducts(ctx context.Context, storeID int, departmentSlug string, page, perPage, offersLimit int) ([]Product, error) {
	return s.api.ListProducts(ctx, storeID, departmentSlug, page, perPage, offersLimit)
}
