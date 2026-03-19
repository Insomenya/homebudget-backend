package models

// Turnover — оборотная ведомость по группе за период.
type Turnover struct {
	Group           SharedGroupWithMembers `json:"group"`
	DateFrom        string                 `json:"date_from"`
	DateTo          string                 `json:"date_to"`
	OpeningBalances []MemberBalance        `json:"opening_balances"`
	Transactions    []Transaction          `json:"transactions"`
	PeriodTotals    []MemberBalance        `json:"period_totals"`
	ClosingBalances []MemberBalance        `json:"closing_balances"`
}