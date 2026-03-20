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
	CategoryID      *int64  `json:"category_id"`
	SharedGroupID   *int64  `json:"shared_group_id"`
	PaidByMemberID  *int64  `json:"paid_by_member_id"`
	LoanID          *int64  `json:"loan_id"`
	Recurrence      string  `json:"recurrence"`
	StartDate       string  `json:"start_date"`
	EndDate         *string `json:"end_date"`
	NextDue         string  `json:"next_due"`
	OriginalDay     int     `json:"original_day"`
	NotifyDaysBefore int    `json:"notify_days_before"`
	OverdueDaysLimit int    `json:"overdue_days_limit"`
	IsActive        bool    `json:"is_active"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type CreatePlannedInput struct {
	Name             string  `json:"name"`
	Amount           float64 `json:"amount"`
	Type             string  `json:"type"`
	CategoryID       *int64  `json:"category_id"`
	SharedGroupID    *int64  `json:"shared_group_id"`
	PaidByMemberID   *int64  `json:"paid_by_member_id"`
	LoanID           *int64  `json:"loan_id"`
	Recurrence       string  `json:"recurrence"`
	StartDate        string  `json:"start_date"`
	EndDate          *string `json:"end_date"`
	NotifyDaysBefore int     `json:"notify_days_before"`
	OverdueDaysLimit int     `json:"overdue_days_limit"`
}

type UpdatePlannedInput struct {
	Name             string  `json:"name"`
	Amount           float64 `json:"amount"`
	Type             string  `json:"type"`
	CategoryID       *int64  `json:"category_id"`
	SharedGroupID    *int64  `json:"shared_group_id"`
	PaidByMemberID   *int64  `json:"paid_by_member_id"`
	LoanID           *int64  `json:"loan_id"`
	Recurrence       string  `json:"recurrence"`
	StartDate        string  `json:"start_date"`
	EndDate          *string `json:"end_date"`
	NotifyDaysBefore int     `json:"notify_days_before"`
	OverdueDaysLimit int     `json:"overdue_days_limit"`
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
	if in.NotifyDaysBefore < 0 {
		in.NotifyDaysBefore = 0
	}
	if in.OverdueDaysLimit < 0 {
		in.OverdueDaysLimit = 0
	}
	if in.SharedGroupID != nil && in.PaidByMemberID == nil {
		return "paid_by_member_id required for shared expense"
	}
	return ""
}

func (in *UpdatePlannedInput) Validate() string {
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
	if in.NotifyDaysBefore < 0 {
		in.NotifyDaysBefore = 0
	}
	if in.OverdueDaysLimit < 0 {
		in.OverdueDaysLimit = 0
	}
	return ""
}

// PlannedReminder — напоминание, видимое в таблице операций.
// Создаётся когда next_due попадает в окно notify_days_before.
// При проводке создаётся транзакция, reminder помечается executed.
type PlannedReminder struct {
	ID              int64   `json:"id"`
	PlannedID       int64   `json:"planned_id"`
	DueDate         string  `json:"due_date"`
	Amount          float64 `json:"amount"`
	TransactionID   *int64  `json:"transaction_id"`
	PrevNextDue     string  `json:"prev_next_due"`
	IsExecuted      bool    `json:"is_executed"`
	CreatedAt       string  `json:"created_at"`
}

// ExecuteReminderInput — данные для проводки напоминания.
type ExecuteReminderInput struct {
	AccountID *int64  `json:"account_id"`
	Amount    float64 `json:"amount"`
	Date      string  `json:"date"`
}

// PlannedForecastItem — для виджета прогноза балансов.
type PlannedForecastItem struct {
	PlannedID   int64   `json:"planned_id"`
	Name        string  `json:"name"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	DueDate     string  `json:"due_date"`
	Enabled     bool    `json:"enabled"`
}

type PlannedForecast struct {
	Items          []PlannedForecastItem  `json:"items"`
	AccountDeltas  map[string]float64     `json:"account_deltas"`
}

// AdvanceDate вычисляет следующую дату платежа.
// Для monthly/quarterly/yearly — привязка к original_day с обработкой коротких месяцев.
func AdvanceDate(current, recurrence string, endDate *string, originalDay int) (next string, active bool) {
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
		t = advanceByMonths(t, 1, originalDay)
	case RecurrenceQuarterly:
		t = advanceByMonths(t, 3, originalDay)
	case RecurrenceYearly:
		t = advanceByMonths(t, 12, originalDay)
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

// advanceByMonths перемещает дату на nMonths вперёд, привязываясь к originalDay.
// Если в целевом месяце меньше дней, ставит на последний день месяца.
func advanceByMonths(t time.Time, nMonths int, originalDay int) time.Time {
	targetYear, targetMonth, _ := t.AddDate(0, nMonths, 0).Date()

	// Сколько дней в целевом месяце
	daysInTarget := daysInMonthFunc(targetYear, targetMonth)

	day := originalDay
	if day > daysInTarget {
		day = daysInTarget
	}
	if day < 1 {
		day = 1
	}

	return time.Date(targetYear, targetMonth, day, 0, 0, 0, 0, time.UTC)
}

func daysInMonthFunc(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// ExtractDay получает день из даты для original_day.
func ExtractDay(dateStr string) int {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 1
	}
	return t.Day()
}