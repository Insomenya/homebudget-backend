package handler

import (
	"net/http"
	"time"

	"homebudget/internal/models"
	"homebudget/internal/repository"
)

type PlannedHandler struct {
	repo   *repository.PlannedRepo
	txRepo *repository.TransactionRepo
}

func (h *PlannedHandler) List(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("all") != "true"
	items, err := h.repo.List(r.Context(), activeOnly)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 200, items)
}

func (h *PlannedHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	p, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if p == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, p)
}

func (h *PlannedHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in models.CreatePlannedInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	p, err := h.repo.Create(r.Context(), in)
	if err != nil {
		if isFKError(err) {
			writeErr(w, 422, "referenced entity not found"); return
		}
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 201, p)
}

func (h *PlannedHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	var in models.UpdatePlannedInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	p, err := h.repo.Update(r.Context(), id, in)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if p == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, p)
}

func (h *PlannedHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeErr(w, 500, err.Error()); return
	}
	w.WriteHeader(204)
}

// Upcoming — GET /api/planned/upcoming?days=30
func (h *PlannedHandler) Upcoming(w http.ResponseWriter, r *http.Request) {
	days := qInt(r, "days", 30)
	items, err := h.repo.Upcoming(r.Context(), days)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 200, items)
}

// Materialize — POST /api/planned/materialize
// Создаёт pending-транзакции для всех отложенных платежей,
// у которых next_due <= сегодня + notify_days.
func (h *PlannedHandler) Materialize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	today := time.Now().Format("2006-01-02")

	active, err := h.repo.List(ctx, true)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}

	var created int
	for _, pt := range active {
		if !pt.IsActive {
			continue
		}
		// Проверяем, что next_due уже наступил или в пределах notify_days
		cutoff := time.Now().AddDate(0, 0, pt.NotifyDays).Format("2006-01-02")
		if pt.NextDue > cutoff {
			continue
		}

		// Проверяем, нет ли уже pending транзакции для этого planned + даты
		exists, err := h.txRepo.ExistsPendingForPlanned(ctx, pt.ID, pt.NextDue)
		if err != nil {
			writeErr(w, 500, err.Error()); return
		}
		if exists {
			continue
		}

		txIn := models.CreateTransactionInput{
			Date:           pt.NextDue,
			Amount:         pt.Amount,
			Description:    pt.Name,
			Type:           pt.Type,
			AccountID:      pt.AccountID,
			CategoryID:     pt.CategoryID,
			SharedGroupID:  pt.SharedGroupID,
			PaidByMemberID: pt.PaidByMemberID,
			IsPending:      true,
			PlannedID:      &pt.ID,
		}

		if msg := txIn.Validate(); msg != "" {
			continue
		}

		if _, err := h.txRepo.Create(ctx, txIn); err != nil {
			continue
		}
		created++

		// Продвигаем next_due
		_ = h.repo.AdvanceNextDue(ctx, pt.ID)
	}

	_ = today
	writeJSON(w, 200, map[string]int{"created": created})
}