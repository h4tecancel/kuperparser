package repository

import (
	"kuperparser/internal/domain/models"
)

type StoreMeta struct {
	ID           int    `json:"id"`
	Name         string `json:"name,omitempty"`
	Address      string `json:"address,omitempty"`
	RetailerName string `json:"retailer_name,omitempty"`
}

type CategoryMeta struct {
	ID   int    `json:"id,omitempty"`
	Slug string `json:"slug,omitempty"`
}

type CategoryResult struct {
	FetchedAt string           `json:"fetched_at"`
	Store     *StoreMeta       `json:"store,omitempty"`
	Category  *CategoryMeta    `json:"category,omitempty"`
	Products  []models.Product `json:"products"`
	Count     int              `json:"count"`
}

type StoresResult struct {
	FetchedAt string      `json:"fetched_at"`
	Stores    []StoreMeta `json:"stores"`
	Count     int         `json:"count"`
}
