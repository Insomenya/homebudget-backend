package models

type BudgetColumn struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ColType   string `json:"col_type"` // "income", "category", "total"
	RefID     *int64 `json:"ref_id"`   // optional FK to categories/members
	SortOrder int    `json:"sort_order"`
}

type CreateBudgetColumnInput struct {
	Name    string `json:"name"`
	ColType string `json:"col_type"`
	RefID   *int64 `json:"ref_id"`
}

type BudgetRow struct {
	ID         int64              `json:"id"`
	Date       string             `json:"date"`
	Label      string             `json:"label"`
	IsExecuted bool               `json:"is_executed"`
	Cells      map[int64]float64  `json:"cells"` // column_id -> amount
}

type CreateBudgetRowInput struct {
	Date  string `json:"date"`
	Label string `json:"label"`
}

type UpdateBudgetCellInput struct {
	RowID    int64   `json:"row_id"`
	ColumnID int64   `json:"column_id"`
	Amount   float64 `json:"amount"`
}

type BudgetTable struct {
	Columns []BudgetColumn `json:"columns"`
	Rows    []BudgetRow    `json:"rows"`
	Total   int            `json:"total"`
	Page    int            `json:"page"`
	Pages   int            `json:"pages"`
}