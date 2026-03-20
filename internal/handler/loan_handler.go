package handler

import (
	"fmt"
	"net/http"

	"homebudget/internal/models"
	"homebudget/internal/repository"
)

type LoanHandler struct {
	repo    *repository.LoanRepo
	account *repository.AccountRepo
	planned *repository.PlannedRepo
	tx      *repository.TransactionRepo
}

func (h *LoanHandler) List(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("all") != "true"
	items, err := h.repo.List(r.Context(), activeOnly)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, items)
}

func (h *LoanHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id")
		return
	}
	l, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if l == nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, l)
}

// Create — создание кредита с опциональными: loan_account, зачисление на счёт, отложенный платёж.
// Body:
//
//	{
//	  ...CreateLoanInput,
//	  "credit_to_account": true,        // зачислить деньги на account_id
//	  "create_planned": true,            // создать отложенный платёж
//	}
func (h *LoanHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		models.CreateLoanInput
		CreditToAccount bool `json:"credit_to_account"`
		CreatePlanned   bool `json:"create_planned"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if msg := body.CreateLoanInput.Validate(); msg != "" {
		writeErr(w, 422, msg)
		return
	}

	ctx := r.Context()

	// 1. Create loan
	loan, err := h.repo.Create(ctx, body.CreateLoanInput)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}

	// 2. Create hidden loan account (всегда)
	// Нужен member_id — берём из default_account или первый member
	var memberID int64 = 1
	if body.DefaultAccountID != nil {
		acc, _ := h.account.GetByID(ctx, *body.DefaultAccountID)
		if acc != nil {
			memberID = acc.MemberID
		}
	}

	loanAcc, err := h.account.Create(ctx, models.CreateAccountInput{
		Name:     fmt.Sprintf("Кредит: %s", loan.Name),
		Type:     "credit",
		Currency: "RUB",
		MemberID: memberID,
		IsHidden: true,
	})
	if err != nil {
		writeErr(w, 500, "loan account: "+err.Error())
		return
	}
	h.repo.SetLoanAccountID(ctx, loan.ID, loanAcc.ID)

	// 3. Credit to account (зачислить деньги)
	if body.CreditToAccount && body.AccountID != nil {
		txIn := models.CreateTransactionInput{
			Date:        loan.StartDate,
			Amount:      loan.Principal,
			Description: fmt.Sprintf("Получение кредита: %s", loan.Name),
			Type:        models.TxTypeIncome,
			AccountID:   body.AccountID,
			CategoryID:  loan.CategoryID,
		}
		h.tx.Create(ctx, txIn)
	}

	// 4. Create planned payment (только для будущих платежей)
	if body.CreatePlanned {
		// Определяем start_date для planned — следующий месяц от start_date, если start_date в прошлом
		plannedStart := loan.StartDate
		today := models.TodayStr()
		if plannedStart <= today {
			// Сдвигаем на первый будущий платёж
			next, _ := models.AdvanceDate(plannedStart, models.RecurrenceMonthly, nil, models.ExtractDay(plannedStart))
			for next <= today {
				n2, _ := models.AdvanceDate(next, models.RecurrenceMonthly, nil, models.ExtractDay(plannedStart))
				if n2 == next {
					break
				}
				next = n2
			}
			plannedStart = next
		}

		endDate := loan.EndDate
		planned, err := h.planned.Create(ctx, models.CreatePlannedInput{
			Name:             fmt.Sprintf("Платёж: %s", loan.Name),
			Amount:           loan.MonthlyPayment,
			Type:             models.TxTypeExpense,
			CategoryID:       loan.CategoryID,
			LoanID:           &loan.ID,
			Recurrence:       models.RecurrenceMonthly,
			StartDate:        plannedStart,
			EndDate:          &endDate,
			NotifyDaysBefore: 3,
			OverdueDaysLimit: 30,
		})
		if err == nil && planned != nil {
			h.repo.SetPlannedID(ctx, loan.ID, planned.ID)
		}
	}

	// Re-fetch with all IDs set
	loan, _ = h.repo.GetByID(ctx, loan.ID)
	writeJSON(w, 201, loan)
}

func (h *LoanHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id")
		return
	}
	var in models.UpdateLoanInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	l, err := h.repo.Update(r.Context(), id, in)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if l == nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, l)
}

func (h *LoanHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id")
		return
	}
	if err := h.repo.DeleteWithCleanup(r.Context(), id, h.planned, h.account); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	w.WriteHeader(204)
}

func (h *LoanHandler) DailySchedule(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id")
		return
	}
	from := qStr(r, "from", "")
	to := qStr(r, "to", "")
	if from == "" || to == "" {
		writeErr(w, 422, "from and to required")
		return
	}
	s, err := h.repo.GetDailySchedule(r.Context(), id, from, to)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if s == nil {
		writeErr(w, 404, "loan not found")
		return
	}
	writeJSON(w, 200, s)
}