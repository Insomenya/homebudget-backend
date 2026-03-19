package models

import (
	"math"
	"time"
)

// ── DB Entity ───────────────────────────────────────

type Loan struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Principal      float64 `json:"principal"`
	AnnualRate     float64 `json:"annual_rate"`
	StartDate      string  `json:"start_date"`
	EndDate        string  `json:"end_date"`
	MonthlyPayment float64 `json:"monthly_payment"`
	AlreadyPaid    float64 `json:"already_paid"`
	AccountID      *int64  `json:"account_id"`
	CategoryID     *int64  `json:"category_id"`
	IsActive       bool    `json:"is_active"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// TermMonths вычисляет срок в месяцах из дат.
func (l *Loan) TermMonths() int {
	start, err1 := time.Parse("2006-01-02", l.StartDate)
	end, err2 := time.Parse("2006-01-02", l.EndDate)
	if err1 != nil || err2 != nil {
		return 0
	}
	months := 0
	cur := start
	for cur.Before(end) {
		cur = cur.AddDate(0, 1, 0)
		months++
	}
	return months
}

type CreateLoanInput struct {
	Name        string  `json:"name"`
	Principal   float64 `json:"principal"`
	AnnualRate  float64 `json:"annual_rate"`
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
	AlreadyPaid float64 `json:"already_paid"`
	AccountID   *int64  `json:"account_id"`
	CategoryID  *int64  `json:"category_id"`
}

type UpdateLoanInput struct {
	Name       string  `json:"name"`
	AnnualRate float64 `json:"annual_rate"`
	AccountID  *int64  `json:"account_id"`
	CategoryID *int64  `json:"category_id"`
	IsActive   bool    `json:"is_active"`
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
	return ""
}

// CalcTermMonths вычисляет количество месяцев между двумя датами.
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

// CalcMonthlyPayment — аннуитетный платёж.
// remainingPrincipal — оставшееся тело (principal - alreadyPaid).
// remainingMonths — оставшиеся месяцы от текущего момента до конца.
func CalcMonthlyPayment(remainingPrincipal, annualRate float64, remainingMonths int) float64 {
	if remainingMonths <= 0 {
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

// CalcMonthlyPaymentForLoan — удобная обёртка для полного кредита.
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

// daysInYear возвращает кол-во дней в году для конкретной даты.
func daysInYear(t time.Time) int {
	start := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(t.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC)
	return int(end.Sub(start).Hours() / 24)
}

var monthNames = []string{
	"", "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
	"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
}

func BuildDailySchedule(loan Loan, payments []Transaction, fromDate, toDate string) *LoanDailySchedule {
	start, _ := time.Parse("2006-01-02", loan.StartDate)
	from, _ := time.Parse("2006-01-02", fromDate)
	to, _ := time.Parse("2006-01-02", toDate)

	// build payment map: date -> total amount
	payMap := make(map[string]float64)
	for _, p := range payments {
		payMap[p.Date] += p.Amount
	}

	debt := loan.Principal - loan.AlreadyPaid
	if debt < 0 {
		debt = 0
	}
	var accumInterest float64
	var totalPaid, totalInterest float64

	// fast-forward from start to fromDate
	for d := start; d.Before(from); d = d.AddDate(0, 0, 1) {
		if debt <= 0.005 {
			break
		}
		dailyRate := loan.AnnualRate / 100.0 / float64(daysInYear(d))
		interest := debt * dailyRate
		accumInterest += interest
		totalInterest += interest

		ds := d.Format("2006-01-02")
		if pmt, ok := payMap[ds]; ok {
			// платёж покрывает сначала накопленные проценты, остаток — в тело
			if accumInterest >= pmt {
				accumInterest -= pmt
			} else {
				principalPart := pmt - accumInterest
				accumInterest = 0
				debt -= principalPart
				if debt < 0 {
					debt = 0
				}
			}
			totalPaid += pmt
		}
	}

	// build visible rows grouped by month
	monthsMap := make(map[string]*LoanMonthGroup)
	var monthOrder []string

	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if debt <= 0.005 && accumInterest <= 0.005 {
			// кредит погашен, но покажем пустые дни? Нет, прерываем.
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
			// досрочный/обычный платёж: сначала проценты, потом тело
			if accumInterest >= pmt {
				accumInterest -= pmt
			} else {
				principalPart := pmt - accumInterest
				accumInterest = 0
				debt -= principalPart
				if debt < 0 {
					debt = 0
				}
			}
			totalPaid += pmt
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

// ── Calculator (stateless) ──────────────────────────

type LoanCalcInput struct {
	Principal    float64 `json:"principal"`
	AnnualRate   float64 `json:"annual_rate"`
	StartDate    string  `json:"start_date"`
	EndDate      string  `json:"end_date"`
	ExtraPayment float64 `json:"extra_payment"`
}

type LoanPayment struct {
	Month         int     `json:"month"`
	Date          string  `json:"date"`
	Payment       float64 `json:"payment"`
	PrincipalPart float64 `json:"principal"`
	InterestPart  float64 `json:"interest"`
	Extra         float64 `json:"extra"`
	Remaining     float64 `json:"remaining"`
	CumPaid       float64 `json:"cumulative_paid"`
	CumInterest   float64 `json:"cumulative_interest"`
}

type LoanCalcResult struct {
	MonthlyPayment   float64       `json:"monthly_payment"`
	TotalPaid        float64       `json:"total_paid"`
	TotalInterest    float64       `json:"total_interest"`
	OverpaymentRatio float64       `json:"overpayment_ratio"`
	EffectiveMonths  int           `json:"effective_months"`
	Schedule         []LoanPayment `json:"schedule"`
}

func (in *LoanCalcInput) Validate() string {
	if in.Principal <= 0 {
		return "principal must be positive"
	}
	if in.AnnualRate < 0 {
		return "annual_rate cannot be negative"
	}
	if in.StartDate == "" || in.EndDate == "" {
		return "start_date and end_date required"
	}
	if _, err := time.Parse("2006-01-02", in.StartDate); err != nil {
		return "start_date must be YYYY-MM-DD"
	}
	if _, err := time.Parse("2006-01-02", in.EndDate); err != nil {
		return "end_date must be YYYY-MM-DD"
	}
	if in.ExtraPayment < 0 {
		return "extra_payment cannot be negative"
	}
	return ""
}

func r2(v float64) float64 { return math.Round(v*100) / 100 }

func (in *LoanCalcInput) Calculate() *LoanCalcResult {
	termMonths := CalcTermMonths(in.StartDate, in.EndDate)
	pmt := CalcMonthlyPayment(in.Principal, in.AnnualRate, termMonths)
	mr := in.AnnualRate / 100.0 / 12.0
	start, _ := time.Parse("2006-01-02", in.StartDate)
	remaining := in.Principal
	var cumPaid, cumInterest float64
	schedule := make([]LoanPayment, 0, termMonths)

	for month := 1; remaining > 0.005; month++ {
		interest := remaining * mr
		principalPart := pmt - interest
		extra := in.ExtraPayment

		totalPrincipal := principalPart + extra
		if totalPrincipal > remaining {
			if principalPart >= remaining {
				principalPart = remaining
				extra = 0
			} else {
				extra = remaining - principalPart
			}
			totalPrincipal = principalPart + extra
		}

		remaining -= totalPrincipal
		if remaining < 0.005 {
			remaining = 0
		}

		payment := interest + totalPrincipal
		cumPaid += payment
		cumInterest += interest

		date := start.AddDate(0, month, 0)

		schedule = append(schedule, LoanPayment{
			Month: month, Date: date.Format("2006-01-02"),
			Payment: r2(payment), PrincipalPart: r2(principalPart),
			InterestPart: r2(interest), Extra: r2(extra),
			Remaining: r2(remaining), CumPaid: r2(cumPaid), CumInterest: r2(cumInterest),
		})

		if remaining == 0 {
			break
		}
	}

	ratio := 0.0
	if in.Principal > 0 {
		ratio = r2(cumPaid / in.Principal)
	}

	return &LoanCalcResult{
		MonthlyPayment: r2(pmt), TotalPaid: r2(cumPaid),
		TotalInterest: r2(cumInterest), OverpaymentRatio: ratio,
		EffectiveMonths: len(schedule), Schedule: schedule,
	}
}