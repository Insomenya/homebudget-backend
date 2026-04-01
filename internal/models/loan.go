// FILE: internal/models/loan.go
package models

import (
	"math"
	"time"
)

// ── DB Entity ───────────────────────────────────────

type Loan struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Principal        float64 `json:"principal"`
	AnnualRate       float64 `json:"annual_rate"`
	StartDate        string  `json:"start_date"`
	EndDate          string  `json:"end_date"`
	MonthlyPayment   float64 `json:"monthly_payment"`
	AlreadyPaid      float64 `json:"already_paid"`
	AccountID        *int64  `json:"account_id"`
	DefaultAccountID *int64  `json:"default_account_id"`
	LoanAccountID    *int64  `json:"loan_account_id"`
	CategoryID       *int64  `json:"category_id"`
	LoanCategoryID   *int64  `json:"loan_category_id"`
	PlannedID        *int64  `json:"planned_id"`
	AccountingStartDate string `json:"accounting_start_date"`
	InitialAccruedInterest float64 `json:"initial_accrued_interest"`
	IsActive         bool    `json:"is_active"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

func (l *Loan) TermMonths() int {
	return CalcTermMonths(l.StartDate, l.EndDate)
}

type CreateLoanInput struct {
	Name             string  `json:"name"`
	Principal        float64 `json:"principal"`
	AnnualRate       float64 `json:"annual_rate"`
	StartDate        string  `json:"start_date"`
	EndDate          string  `json:"end_date"`
	AlreadyPaid      float64 `json:"already_paid"`
	AccountID        *int64  `json:"account_id"`
	DefaultAccountID *int64  `json:"default_account_id"`
	CategoryID       *int64  `json:"category_id"`
	AccountingStartDate *string `json:"accounting_start_date"`
	InitialAccruedInterest float64 `json:"initial_accrued_interest"`
}

type UpdateLoanInput struct {
	Name             string  `json:"name"`
	AnnualRate       float64 `json:"annual_rate"`
	DefaultAccountID *int64  `json:"default_account_id"`
	CategoryID       *int64  `json:"category_id"`
	IsActive         bool    `json:"is_active"`
}

func (in *CreateLoanInput) Validate() string {
	if in.Name == "" {
		return "name is required"
	}
	if in.Principal <= 0 {
		return "principal must be positive"
	}
	if in.AnnualRate < 0 {
		return "annual_rate cannot be negative"
	}
	if in.StartDate == "" {
		return "start_date is required"
	}
	if _, err := time.Parse("2006-01-02", in.StartDate); err != nil {
		return "start_date must be YYYY-MM-DD"
	}
	if in.EndDate == "" {
		return "end_date is required"
	}
	endT, err := time.Parse("2006-01-02", in.EndDate)
	if err != nil {
		return "end_date must be YYYY-MM-DD"
	}
	startT, _ := time.Parse("2006-01-02", in.StartDate)
	if !endT.After(startT) {
		return "end_date must be after start_date"
	}
	if in.AccountingStartDate != nil && *in.AccountingStartDate != "" {
		ast, err := time.Parse("2006-01-02", *in.AccountingStartDate)
		if err != nil {
			return "accounting_start_date must be YYYY-MM-DD"
		}
		if ast.Before(startT) {
			return "accounting_start_date must be >= start_date"
		}
	}
	if in.InitialAccruedInterest < 0 {
		return "initial_accrued_interest cannot be negative"
	}
	return ""
}

// ── Helpers ─────────────────────────────────────────

func TodayStr() string {
	return time.Now().Format("2006-01-02")
}

func CalcTermMonths(startDate, endDate string) int {
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	months := 0
	cur := start
	for cur.Before(end) {
		cur = cur.AddDate(0, 1, 0)
		months++
	}
	if months == 0 {
		months = 1
	}
	return months
}

func CalcMonthlyPayment(remainingPrincipal, annualRate float64, remainingMonths int) float64 {
	if remainingMonths <= 0 || remainingPrincipal <= 0 {
		return 0
	}
	mr := annualRate / 100.0 / 12.0
	n := float64(remainingMonths)
	if mr < 1e-10 {
		return math.Ceil(remainingPrincipal/n*100) / 100
	}
	rn := math.Pow(1+mr, n)
	return math.Ceil(remainingPrincipal*mr*rn/(rn-1)*100) / 100
}

func CalcMonthlyPaymentForLoan(principal, alreadyPaid, annualRate float64, startDate, endDate string) float64 {
	remaining := principal - alreadyPaid
	if remaining <= 0 {
		return 0
	}
	months := CalcTermMonths(startDate, endDate)
	return CalcMonthlyPayment(remaining, annualRate, months)
}

// ── Daily schedule ──────────────────────────────────

type LoanDayRow struct {
	Date          string  `json:"date"`
	Day           int     `json:"day"`
	Debt          float64 `json:"debt"`
	DailyInterest float64 `json:"daily_interest"`
	AccumInterest float64 `json:"accrued_interest"`
	Payment       float64 `json:"payment"`
	IsPaymentDay  bool    `json:"is_payment_day"`
}

type LoanMonthGroup struct {
	Month string       `json:"month"`
	Label string       `json:"label"`
	Days  []LoanDayRow `json:"days"`
}

type LoanDailySchedule struct {
	Loan          Loan             `json:"loan"`
	CurrentDebt   float64          `json:"current_debt"`
	TotalPaid     float64          `json:"total_paid"`
	TotalInterest float64          `json:"total_interest"`
	Months        []LoanMonthGroup `json:"months"`
}

func daysInYear(t time.Time) int {
	start := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(t.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC)
	return int(end.Sub(start).Hours() / 24)
}

var monthNames = []string{
	"", "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
	"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
}

// BuildDailySchedule builds a day-by-day loan schedule.
// Interest accrues from startDate. Payments reduce accrued interest first, then principal.
// Multiple payments on the same day are summed.
func BuildDailySchedule(loan Loan, payments []Transaction, fromDate, toDate string) *LoanDailySchedule {
	start, _ := time.Parse("2006-01-02", loan.StartDate)
	effectiveStart := start
	if loan.AccountingStartDate != "" {
		if accStart, err := time.Parse("2006-01-02", loan.AccountingStartDate); err == nil && accStart.After(effectiveStart) {
			effectiveStart = accStart
		}
	}
	from, _ := time.Parse("2006-01-02", fromDate)
	to, _ := time.Parse("2006-01-02", toDate)
	if from.Before(effectiveStart) {
		from = effectiveStart
	}
	if to.Before(from) {
		to = from
	}

	// Build payment map: sum all payments per day
	payMap := make(map[string]float64)
	for _, p := range payments {
		payMap[p.Date] += p.Amount
	}

	debt := loan.Principal - loan.AlreadyPaid
	if debt < 0 {
		debt = 0
	}
	accumInterest := loan.InitialAccruedInterest
	var totalPaid, totalInterest float64

	// Process days before the visible range (from accounting start to from-1)
	for d := effectiveStart; d.Before(from); d = d.AddDate(0, 0, 1) {
		if debt <= 0.005 {
			break
		}
		dailyRate := loan.AnnualRate / 100.0 / float64(daysInYear(d))
		interest := debt * dailyRate
		accumInterest += interest
		totalInterest += interest

		ds := d.Format("2006-01-02")
		if pmt, ok := payMap[ds]; ok && pmt > 0 {
			totalPaid += pmt
			if pmt <= accumInterest {
				accumInterest -= pmt
			} else {
				principalPart := pmt - accumInterest
				accumInterest = 0
				debt -= principalPart
				if debt < 0 {
					debt = 0
				}
			}
		}
	}

	monthsMap := make(map[string]*LoanMonthGroup)
	var monthOrder []string

	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if debt <= 0.005 && accumInterest <= 0.005 {
			break
		}

		dailyRate := loan.AnnualRate / 100.0 / float64(daysInYear(d))
		interest := debt * dailyRate
		accumInterest += interest
		totalInterest += interest

		ds := d.Format("2006-01-02")
		pmt := payMap[ds]
		isPayDay := pmt > 0

		if isPayDay {
			totalPaid += pmt
			if pmt <= accumInterest {
				accumInterest -= pmt
			} else {
				principalPart := pmt - accumInterest
				accumInterest = 0
				debt -= principalPart
				if debt < 0 {
					debt = 0
				}
			}
		}

		monthKey := d.Format("2006-01")
		if _, ok := monthsMap[monthKey]; !ok {
			label := monthNames[int(d.Month())] + " " + d.Format("2006")
			mg := &LoanMonthGroup{Month: monthKey, Label: label}
			monthsMap[monthKey] = mg
			monthOrder = append(monthOrder, monthKey)
		}

		monthsMap[monthKey].Days = append(monthsMap[monthKey].Days, LoanDayRow{
			Date:          ds,
			Day:           d.Day(),
			Debt:          math.Round(debt*100) / 100,
			DailyInterest: math.Round(interest*100) / 100,
			AccumInterest: math.Round(accumInterest*100) / 100,
			Payment:       math.Round(pmt*100) / 100,
			IsPaymentDay:  isPayDay,
		})
	}

	months := make([]LoanMonthGroup, 0, len(monthOrder))
	for _, k := range monthOrder {
		months = append(months, *monthsMap[k])
	}

	return &LoanDailySchedule{
		Loan:          loan,
		CurrentDebt:   math.Round(debt*100) / 100,
		TotalPaid:     math.Round(totalPaid*100) / 100,
		TotalInterest: math.Round(totalInterest*100) / 100,
		Months:        months,
	}
}