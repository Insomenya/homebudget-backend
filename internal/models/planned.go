package models

import "time"

const (
	RecurrenceOnce      = "once"
	RecurrenceWeekly    = "weekly"
	RecurrenceBiweekly  = "biweekly"
	RecurrenceMonthly   = "monthly"
	RecurrenceQuarterly = "quarterly"
	RecurrenceYearly    = "yearly"
)

var validRecurrences = map[string]bool{
	RecurrenceOnce: true, RecurrenceWeekly: true,
	RecurrenceBiweekly: true, RecurrenceMonthly: true,
	RecurrenceQuarterly: true, RecurrenceYearly: true,
}

type PlannedTransaction struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Amount          float64 `json:"amount"`
	Type            string  `json:"type"`
	AccountID       *int64  `json:"account_id"`
	CategoryID      *int64  `json:"category_id"`
	SharedGroupID   *int64  `json:"shared_group_id"`
	PaidByMemberID  *int64  `json:"paid_by_member_id"`
	Recurrence      string  `json:"recurrence"`
	StartDate       string  `json:"start_date"`
	EndDate         *string `json:"end_date"`
	NextDue         string  `json:"next_due"`
	NotifyDays      int     `json:"notify_days"`
	IsAuto          bool    `json:"is_auto"`
	IsActive        bool    `json:"is_active"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type CreatePlannedInput struct {
	Name            string  `json:"name"`
	Amount          float64 `json:"amount"`
	Type            string  `json:"type"`
	AccountID       *int64  `json:"account_id"`
	CategoryID      *int64  `json:"category_id"`
	SharedGroupID   *int64  `json:"shared_group_id"`
	PaidByMemberID  *int64  `json:"paid_by_member_id"`
	Recurrence      string  `json:"recurrence"`
	StartDate       string  `json:"start_date"`
	EndDate         *string `json:"end_date"`
	NotifyDays      int     `json:"notify_days"`
	IsAuto          bool    `json:"is_auto"`
}

type UpdatePlannedInput = CreatePlannedInput

type ExecutePlannedInput struct {
	Date string `json:"date"`
}

func (in *CreatePlannedInput) Validate() string {
	if in.Name == "" {
		return "name is required"
	}
	if in.Amount <= 0 {
		return "amount must be positive"
	}
	switch in.Type {
	case TxTypeExpense, TxTypeIncome, TxTypeTransfer:
	case "":
		in.Type = TxTypeExpense
	default:
		return "invalid type"
	}
	if in.Recurrence == "" {
		in.Recurrence = RecurrenceMonthly
	}
	if !validRecurrences[in.Recurrence] {
		return "invalid recurrence"
	}
	if in.StartDate == "" {
		return "start_date is required"
	}
	if _, err := time.Parse("2006-01-02", in.StartDate); err != nil {
		return "start_date must be YYYY-MM-DD"
	}
	if in.EndDate != nil {
		if _, err := time.Parse("2006-01-02", *in.EndDate); err != nil {
			return "end_date must be YYYY-MM-DD"
		}
	}
	if in.NotifyDays < 0 {
		in.NotifyDays = 0
	}
	return ""
}

// AdvanceDate вычисляет следующую дату и признак активности.
func AdvanceDate(current, recurrence string, endDate *string) (next string, active bool) {
	t, err := time.Parse("2006-01-02", current)
	if err != nil {
		return current, false
	}

	switch recurrence {
	case RecurrenceOnce:
		return current, false
	case RecurrenceWeekly:
		t = t.AddDate(0, 0, 7)
	case RecurrenceBiweekly:
		t = t.AddDate(0, 0, 14)
	case RecurrenceMonthly:
		t = t.AddDate(0, 1, 0)
	case RecurrenceQuarterly:
		t = t.AddDate(0, 3, 0)
	case RecurrenceYearly:
		t = t.AddDate(1, 0, 0)
	}

	next = t.Format("2006-01-02")
	active = true

	if endDate != nil {
		if next > *endDate {
			active = false
		}
	}
	return
}