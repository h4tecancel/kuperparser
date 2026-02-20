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

type categoriesResp struct {
	Categories []responses.Category `json:"categories"`
}

func (c *Client) ListCategories(ctx context.Context, storeID int) ([]responses.Category, error) {
	req, err := c.newReq(ctx, http.MethodGet, fmt.Sprintf("/api/v3/stores/%d/categories", storeID))
	if err != nil {
		return nil, err
	}

	resp, err := c.Doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ListCategories: status=%d body=%s",
			resp.StatusCode, strings.TrimSpace(string(b[:min(len(b), 4096)])))
	}

	var out categoriesResp
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}

	return out.Categories, nil
}
