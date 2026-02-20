package responses

type Category struct {
	ID            int        `json:"id"`
	ParentID      int        `json:"parent_id"`
	Type          string     `json:"type"`
	Name          string     `json:"name"`
	Slug          string     `json:"slug"`
	ProductsCount int        `json:"products_count"`
	CategoryType  string     `json:"category_type"`
	HasChildren   bool       `json:"has_children"`
	Children      []Category `json:"children"`
}
