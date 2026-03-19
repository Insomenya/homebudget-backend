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
	TermMonths     int     `json:"term_months"`
	StartDate      string  `json:"start_date"`
	MonthlyPayment float64 `json:"monthly_payment"`
	AlreadyPaid    float64 `json:"already_paid"`
	AccountID      *int64  `json:"account_id"`
	CategoryID     *int64  `json:"category_id"`
	IsActive       bool    `json:"is_active"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type CreateLoanInput struct {
	Name        string  `json:"name"`
	Principal   float64 `json:"principal"`
	AnnualRate  float64 `json:"annual_rate"`
	TermMonths  int     `json:"term_months"`
	StartDate   string  `json:"start_date"`
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
	if in.TermMonths <= 0 || in.TermMonths > 600 {
		return "term_months must be 1..600"
	}
	if in.StartDate == "" {
		return "start_date is required"
	}
	if _, err := time.Parse("2006-01-02", in.StartDate); err != nil {
		return "start_date must be YYYY-MM-DD"
	}
	return ""
}

func CalcMonthlyPayment(principal, annualRate float64, termMonths int) float64 {
	mr := annualRate / 100.0 / 12.0
	n := float64(termMonths)
	if mr < 1e-10 {
		return math.Ceil(principal / n * 100) / 100
	}
	rn := math.Pow(1+mr, n)
	return math.Ceil(principal * mr * rn / (rn - 1) * 100) / 100
}

// ── Daily schedule ──────────────────────────────────

type LoanDayRow struct {
	Date            string  `json:"date"`
	Debt            float64 `json:"debt"`
	DailyInterest   float64 `json:"daily_interest"`
	AccumInterest   float64 `json:"accrued_interest"`
	Payment         float64 `json:"payment"`
	IsPaymentDay    bool    `json:"is_payment_day"`
}

type LoanDailySchedule struct {
	Loan       Loan         `json:"loan"`
	CurrentDebt float64     `json:"current_debt"`
	TotalPaid   float64     `json:"total_paid"`
	TotalInterest float64   `json:"total_interest"`
	Days       []LoanDayRow `json:"days"`
}

func BuildDailySchedule(loan Loan, payments []Transaction, fromDate, toDate string) *LoanDailySchedule {
	start, _ := time.Parse("2006-01-02", loan.StartDate)
	from, _ := time.Parse("2006-01-02", fromDate)
	to, _ := time.Parse("2006-01-02", toDate)

	dailyRate := loan.AnnualRate / 100.0 / 365.0

	// build payment map: date -> total amount
	payMap := make(map[string]float64)
	for _, p := range payments {
		payMap[p.Date] += p.Amount
	}

	debt := loan.Principal - loan.AlreadyPaid
	var accumInterest float64
	var totalPaid, totalInterest float64

	// fast-forward from start to fromDate
	for d := start; d.Before(from); d = d.AddDate(0, 0, 1) {
		interest := debt * dailyRate
		accumInterest += interest
		totalInterest += interest

		ds := d.Format("2006-01-02")
		if pmt, ok := payMap[ds]; ok {
			// payment covers accrued interest first, rest goes to principal
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

	// build visible rows
	days := make([]LoanDayRow, 0)
	for d := from; !d.After(to) && debt > 0.01; d = d.AddDate(0, 0, 1) {
		interest := debt * dailyRate
		accumInterest += interest
		totalInterest += interest

		ds := d.Format("2006-01-02")
		pmt := payMap[ds]
		isPayDay := pmt > 0

		if isPayDay {
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

		days = append(days, LoanDayRow{
			Date:          ds,
			Debt:          math.Round(debt*100) / 100,
			DailyInterest: math.Round(interest*100) / 100,
			AccumInterest: math.Round(accumInterest*100) / 100,
			Payment:       math.Round(pmt*100) / 100,
			IsPaymentDay:  isPayDay,
		})
	}

	return &LoanDailySchedule{
		Loan:          loan,
		CurrentDebt:   math.Round(debt*100) / 100,
		TotalPaid:     math.Round(totalPaid*100) / 100,
		TotalInterest: math.Round(totalInterest*100) / 100,
		Days:          days,
	}
}

// ── Calculator (stateless, kept for tools endpoint) ──

type LoanCalcInput struct {
	Principal    float64 `json:"principal"`
	AnnualRate   float64 `json:"annual_rate"`
	TermMonths   int     `json:"term_months"`
	StartDate    string  `json:"start_date"`
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
	if in.TermMonths <= 0 || in.TermMonths > 600 {
		return "term_months must be 1..600"
	}
	if in.StartDate == "" {
		in.StartDate = time.Now().Format("2006-01-02")
	}
	if in.ExtraPayment < 0 {
		return "extra_payment cannot be negative"
	}
	return ""
}

func r2(v float64) float64 { return math.Round(v*100) / 100 }

func (in *LoanCalcInput) Calculate() *LoanCalcResult {
	pmt := CalcMonthlyPayment(in.Principal, in.AnnualRate, in.TermMonths)
	mr := in.AnnualRate / 100.0 / 12.0
	start, _ := time.Parse("2006-01-02", in.StartDate)
	remaining := in.Principal
	var cumPaid, cumInterest float64
	schedule := make([]LoanPayment, 0, in.TermMonths)

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

	return &LoanCalcResult{
		MonthlyPayment: r2(pmt), TotalPaid: r2(cumPaid),
		TotalInterest: r2(cumInterest), OverpaymentRatio: r2(cumPaid / in.Principal),
		EffectiveMonths: len(schedule), Schedule: schedule,
	}
}