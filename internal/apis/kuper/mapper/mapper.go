package mapper

import (
	"fmt"
	"strings"

	"kuperparser/internal/apis/kuper"
	"kuperparser/internal/domain/models"
)

func FromProduct(baseURL string, p kuper.Product) models.Product {
	return models.Product{
		Name:  extractName(p),
		Price: extractPrice(p),
		URL:   extractURL(baseURL, p),
	}
}

func extractName(p kuper.Product) string {
	if v, ok := asString(p.Raw["name"]); ok && v != "" {
		return v
	}
	if v, ok := asString(p.Raw["title"]); ok && v != "" {
		return v
	}
	return ""
}

func extractURL(baseURL string, p kuper.Product) string {
	if v, ok := asString(p.Raw["canonical_url"]); ok && strings.HasPrefix(v, "http") {
		return v
	}
	if v, ok := asString(p.Raw["url"]); ok && strings.HasPrefix(v, "http") {
		return v
	}

	if v, ok := asString(p.Raw["permalink"]); ok && v != "" {
		if strings.HasPrefix(v, "http") {
			return v
		}
		if strings.HasPrefix(v, "/") {
			return baseURL + v
		}
		return baseURL + "/" + v
	}

	return ""
}

func extractPrice(p kuper.Product) string {
	if v, ok := asString(p.Raw["price"]); ok && v != "" {
		return normalizePrice(v)
	}
	if v, ok := asNumberString(p.Raw["price"]); ok && v != "" {
		return normalizePrice(v)
	}

	if arr, ok := p.Raw["offers"].([]any); ok && len(arr) > 0 {
		if m, ok := arr[0].(map[string]any); ok {
			if v, ok := asNumberString(m["price"]); ok && v != "" {
				return normalizePrice(v)
			}
			if pm, ok := m["price"].(map[string]any); ok {
				if v, ok := asNumberString(pm["amount"]); ok && v != "" {
					return normalizePrice(v)
				}
				if v, ok := asNumberString(pm["value"]); ok && v != "" {
					return normalizePrice(v)
				}
			}
		}
	}

	if v, ok := asNumberString(p.Raw["current_price"]); ok && v != "" {
		return normalizePrice(v)
	}
	if v, ok := asNumberString(p.Raw["price_current"]); ok && v != "" {
		return normalizePrice(v)
	}

	return ""
}

func normalizePrice(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\u00A0", " ")
	s = strings.ReplaceAll(s, "₽", "")
	s = strings.ReplaceAll(s, "руб.", "")
	s = strings.ReplaceAll(s, "руб", "")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")
	return s
}

func asString(v any) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	return s, true
}

func asNumberString(v any) (string, bool) {
	switch t := v.(type) {
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t)), true
		}
		return fmt.Sprintf("%v", t), true
	case int:
		return fmt.Sprintf("%d", t), true
	case int64:
		return fmt.Sprintf("%d", t), true
	case string:
		if t != "" {
			return t, true
		}
	}
	return "", false
}
