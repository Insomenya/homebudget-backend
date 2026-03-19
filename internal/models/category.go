package models

const (
	CategoryTypeExpense = "expense"
	CategoryTypeIncome  = "income"
)

type Category struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Icon       string `json:"icon"`
	ParentID   *int64 `json:"parent_id"`
	SortOrder  int    `json:"sort_order"`
	IsArchived bool   `json:"is_archived"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type CreateCategoryInput struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Icon      string `json:"icon"`
	ParentID  *int64 `json:"parent_id"`
	SortOrder int    `json:"sort_order"`
}

type UpdateCategoryInput struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Icon       string `json:"icon"`
	ParentID   *int64 `json:"parent_id"`
	SortOrder  int    `json:"sort_order"`
	IsArchived bool   `json:"is_archived"`
}

func validCatType(t string) bool {
	return t == CategoryTypeExpense || t == CategoryTypeIncome
}

func (in *CreateCategoryInput) Validate() string {
	if in.Name == "" {
		return "name is required"
	}
	if in.Type == "" {
		in.Type = CategoryTypeExpense
	}
	if !validCatType(in.Type) {
		return "type must be 'expense' or 'income'"
	}
	return ""
}

func (in *UpdateCategoryInput) Validate() string {
	if in.Name == "" {
		return "name is required"
	}
	if !validCatType(in.Type) {
		return "type must be 'expense' or 'income'"
	}
	return ""
}