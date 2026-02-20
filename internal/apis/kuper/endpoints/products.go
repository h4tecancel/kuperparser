package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"kuperparser/internal/apis/kuper/responses"
)

func (c *Client) ListProducts(ctx context.Context, storeID int, departmentSlug string, page, perPage, offersLimit int) ([]responses.Product, error) {
	path := fmt.Sprintf(
		"/api/v3/stores/%d/departments/%s?offers_limit=%d&page=%d&per_page=%d",
		storeID, departmentSlug, offersLimit, page, perPage,
	)

	req, err := c.newReq(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	resp, err := c.Doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ParseAPIError(resp.StatusCode, []byte(strings.TrimSpace(string(b))))
	}

	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("ListProducts: bad json body=%s", string(b[:min(len(b), 1024)]))
	}

	if deps, ok := raw["departments"].([]any); ok {
		out := make([]responses.Product, 0, 128)

		for _, d := range deps {
			dep, ok := d.(map[string]any)
			if !ok {
				continue
			}

			depSlug := pickString(dep, "slug", "department_slug", "category_slug")
			prods, ok := dep["products"].([]any)
			if !ok || len(prods) == 0 {
				continue
			}

			out = append(out, toProducts(prods, depSlug)...)
		}

		if len(out) > 0 {
			return out, nil
		}
	}

	// deals[]
	if arr, ok := raw["deals"].([]any); ok && len(arr) > 0 {
		return toProducts(arr, ""), nil
	}

	// products[] / items[]
	if arr, ok := raw["products"].([]any); ok {
		return toProducts(arr, ""), nil
	}
	if arr, ok := raw["items"].([]any); ok {
		return toProducts(arr, ""), nil
	}

	// data.products[]
	if data, ok := raw["data"].(map[string]any); ok {
		if arr, ok := data["products"].([]any); ok {
			return toProducts(arr, ""), nil
		}
	}

	if code, ok := raw["code"]; ok {
		msg, _ := raw["message"].(string)
		return nil, fmt.Errorf("ListProducts: api error code=%v message=%s", code, msg)
	}

	return []responses.Product{}, nil
}

func toProducts(arr []any, depSlug string) []responses.Product {
	res := make([]responses.Product, 0, len(arr))
	for _, it := range arr {
		if m, ok := it.(map[string]any); ok {
			// метка, чтобы выше можно было фильтровать leaf-категории
			if depSlug != "" {
				m["_department_slug"] = depSlug
			}
			res = append(res, responses.Product{Raw: m})
		}
	}
	return res
}

func pickString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
