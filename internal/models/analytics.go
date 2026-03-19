package models

// ── Круговая диаграмма по категориям ────────────────────────────

type CategorySlice struct {
	CategoryID   int64   `json:"category_id"`
	CategoryName string  `json:"category_name"`
	CategoryIcon string  `json:"category_icon"`
	ParentID     *int64  `json:"parent_id"`
	Amount       float64 `json:"amount"`
	Percentage   float64 `json:"percentage"`
}

type CategoryBreakdown struct {
	Type  string          `json:"type"`
	Total float64         `json:"total"`
	Items []CategorySlice `json:"items"`
}

// ── Динамика по времени ─────────────────────────────────────────

type TrendPoint struct {
	Period   string  `json:"period"`
	Income   float64 `json:"income"`
	Expenses float64 `json:"expenses"`
	Net      float64 `json:"net"`
}

type TrendData struct {
	Granularity string       `json:"granularity"`
	DateFrom    string       `json:"date_from"`
	DateTo      string       `json:"date_to"`
	Points      []TrendPoint `json:"points"`
}

// ── Фильтры аналитики ──────────────────────────────────────────

type AnalyticsFilter struct {
	DateFrom      string
	DateTo        string
	Type          string // expense | income
	AccountID     *int64
	CategoryID    *int64
	SharedGroupID *int64
	Granularity   string // day | week | month | year
}