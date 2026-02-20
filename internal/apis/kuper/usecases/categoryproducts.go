package usecases

import (
	"context"
	"errors"
	"fmt"
	"kuperparser/internal/apis/kuper"
	"kuperparser/internal/apis/kuper/mapper"

	"kuperparser/internal/domain/models"
	"log/slog"
)

type CategoryProductsService struct {
	kuper       kuper.KuperService
	baseURL     string
	log         *slog.Logger
	perPage     int
	offersLimit int
	maxPages    int
}

func NewCategoryProductsService(
	kuperSvc kuper.KuperService,
	baseURL string,
	logger *slog.Logger,
	perPage int,
	offersLimit int,
	maxPages int,
) *CategoryProductsService {
	if logger == nil {
		logger = slog.Default()
	}
	if perPage <= 0 {
		perPage = 5
	}
	if perPage > 5 {
		perPage = 5
	}
	if offersLimit <= 0 {
		offersLimit = 10
	}
	if maxPages <= 0 {
		maxPages = 500
	}

	return &CategoryProductsService{
		kuper:       kuperSvc,
		baseURL:     baseURL,
		log:         logger,
		perPage:     perPage,
		offersLimit: offersLimit,
		maxPages:    maxPages,
	}
}

func (s *CategoryProductsService) ResolveDepartmentAndLeafSlug(ctx context.Context, storeID, categoryID int) (departmentSlug string, leafSlug string, err error) {
	if storeID <= 0 {
		return "", "", fmt.Errorf("storeID must be > 0")
	}
	if categoryID <= 0 {
		return "", "", fmt.Errorf("categoryID must be > 0")
	}

	cats, err := s.kuper.ListCategories(ctx, storeID)
	if err != nil {
		return "", "", fmt.Errorf("list categories: %w", err)
	}

	path, ok := findPathByID(cats, categoryID)
	if !ok || len(path) == 0 {
		return "", "", fmt.Errorf("categoryID=%d not found for storeID=%d", categoryID, storeID)
	}

	target := path[len(path)-1]
	if target.Slug == "" {
		return "", "", fmt.Errorf("categoryID=%d found but slug empty", categoryID)
	}

	isDept := func(c kuper.Category) bool {
		return c.HasChildren || len(c.Children) > 0
	}

	if isDept(target) {
		return target.Slug, "", nil
	}

	for i := len(path) - 2; i >= 0; i-- {
		if isDept(path[i]) && path[i].Slug != "" {
			return path[i].Slug, target.Slug, nil
		}
	}

	return target.Slug, "", nil
}

func findPathByID(cats []kuper.Category, id int) ([]kuper.Category, bool) {
	for _, c := range cats {
		if c.ID == id {
			return []kuper.Category{c}, true
		}
		if len(c.Children) > 0 {
			if p, ok := findPathByID(c.Children, id); ok {
				return append([]kuper.Category{c}, p...), true
			}
		}
	}
	return nil, false
}

func (s *CategoryProductsService) GetByCategoryID(ctx context.Context, storeID int, categoryID int) ([]models.Product, string, error) {
	deptSlug, leafSlug, err := s.ResolveDepartmentAndLeafSlug(ctx, storeID, categoryID)
	if err != nil {
		return nil, "", err
	}

	products, err := s.GetByDepartmentSlug(ctx, storeID, deptSlug, leafSlug)
	if err != nil {
		used := deptSlug
		if leafSlug != "" {
			used = leafSlug
		}
		return nil, used, err
	}

	used := deptSlug
	if leafSlug != "" {
		used = leafSlug
	}
	return products, used, nil

}

func (s *CategoryProductsService) GetByDepartmentSlug(
	ctx context.Context,
	storeID int,
	departmentSlug string,
	onlyChildSlug string,
) ([]models.Product, error) {
	if storeID <= 0 {
		return nil, fmt.Errorf("storeID must be > 0")
	}
	if departmentSlug == "" {
		return nil, fmt.Errorf("departmentSlug must not be empty")
	}

	s.log.Info("fetch category products",
		"store_id", storeID,
		"department_slug", departmentSlug,
		"only_child_slug", onlyChildSlug,
		"per_page", s.perPage,
		"offers_limit", s.offersLimit,
	)

	out := make([]models.Product, 0, 128)
	total := 0

	for page := 1; page <= s.maxPages; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		raw, err := s.kuper.ListProducts(ctx, storeID, departmentSlug, page, s.perPage, s.offersLimit)
		if err != nil {
			return nil, fmt.Errorf("list products slug=%s page=%d: %w", departmentSlug, page, err)
		}

		rawLen := len(raw)
		if rawLen == 0 {
			break
		}
		if onlyChildSlug != "" {
			filtered := raw[:0]
			for _, p := range raw {
				if p.Raw == nil {
					continue
				}
				if v, ok := p.Raw["_department_slug"].(string); ok && v == onlyChildSlug {
					filtered = append(filtered, p)
				}
			}
			raw = filtered
		}

		for _, p := range raw {
			dp := mapper.FromProduct(s.baseURL, p)
			if dp.Name == "" && dp.URL == "" && dp.Price == "" {
				continue
			}
			out = append(out, dp)
			total++
			if total > 200_000 {
				return nil, errors.New("too many products parsed: possible infinite pagination")
			}
		}

		if rawLen < s.perPage {
			break
		}
	}

	s.log.Info("category products fetched",
		"store_id", storeID,
		"department_slug", departmentSlug,
		"only_child_slug", onlyChildSlug,
		"count", len(out),
	)

	return out, nil
}

func (s *CategoryProductsService) GetBySlug(ctx context.Context, storeID int, slug string) ([]models.Product, error) {
	return s.GetByDepartmentSlug(ctx, storeID, slug, "")
}
