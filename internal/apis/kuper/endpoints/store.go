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

type storeResp struct {
	Store struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Location struct {
			FullAddress string `json:"full_address"`
			City        string `json:"city"`
			Street      string `json:"street"`
			Building    string `json:"building"`
		} `json:"location"`
		Retailer struct {
			Name string `json:"name"`
		} `json:"retailer"`
	} `json:"store"`
}

func (c *Client) GetStore(ctx context.Context, storeID int) (responses.StoreInfo, error) {
	req, err := c.newReq(ctx, http.MethodGet, fmt.Sprintf("/api/stores/%d", storeID))
	if err != nil {
		return responses.StoreInfo{}, err
	}

	resp, err := c.Doer.Do(req)
	if err != nil {
		return responses.StoreInfo{}, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return responses.StoreInfo{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return responses.StoreInfo{}, &APIError{
			Code:    500,
			Status:  resp.StatusCode,
			Message: "status code of storeinfo != status ok",
			Body:    string(b),
		}
	}

	var out storeResp
	if err := json.Unmarshal(b, &out); err != nil {
		return responses.StoreInfo{}, err
	}

	addr := out.Store.Location.FullAddress
	if addr == "" {
		addr = strings.TrimSpace(fmt.Sprintf("%s, %s %s",
			out.Store.Location.City, out.Store.Location.Street, out.Store.Location.Building))
	}

	name := out.Store.Name
	if name == "" {
		name = out.Store.FullName
	}

	return responses.StoreInfo{
		StoreID:      out.Store.ID,
		StoreName:    name,
		StoreAddress: addr,
		RetailerName: out.Store.Retailer.Name,
	}, nil
}
