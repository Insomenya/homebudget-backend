package models

type AccountBalance struct {
	Account
	CurrentBalance float64 `json:"current_balance"`
}

type CategoryTotal struct {
	CategoryID   int64   `json:"category_id"`
	CategoryName string  `json:"category_name"`
	CategoryIcon string  `json:"category_icon"`
	Amount       float64 `json:"amount"`
}

type PeriodSummary struct {
	TotalIncome   float64         `json:"total_income"`
	TotalExpenses float64         `json:"total_expenses"`
	ByCategory    []CategoryTotal `json:"by_category"`
}

type GroupSettlementSummary struct {
	GroupID   int64  `json:"group_id"`
	GroupName string `json:"group_name"`
	GroupIcon string `json:"group_icon"`
	Debts     []Debt `json:"debts"`
}

type Dashboard struct {
	Accounts     []AccountBalance        `json:"accounts"`
	CurrentMonth PeriodSummary           `json:"current_month"`
	Settlements  []GroupSettlementSummary `json:"settlements"`
	Recent       []Transaction           `json:"recent"`
	Upcoming     []PlannedTransaction    `json:"upcoming"`
	Pending      []Transaction           `json:"pending"`
}