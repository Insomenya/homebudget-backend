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

// Execute — POST /api/planned/{id}/execute
// Создаёт реальную транзакцию и продвигает next_due.
func (h *PlannedHandler) Execute(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}

	pt, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if pt == nil {
		writeErr(w, 404, "not found"); return
	}
	if !pt.IsActive {
		writeErr(w, 422, "planned transaction is inactive"); return
	}

	// Опциональный override даты
	var body models.ExecutePlannedInput
	_ = readJSON(r, &body) // ошибка парсинга — не критична, используем next_due

	date := pt.NextDue
	if body.Date != "" {
		if _, err := time.Parse("2006-01-02", body.Date); err == nil {
			date = body.Date
		}
	}

	txIn := models.CreateTransactionInput{
		Date:            date,
		Amount:          pt.Amount,
		Description:     pt.Name,
		Type:            pt.Type,
		AccountID:       pt.AccountID,
		CategoryID:      pt.CategoryID,
		SharedGroupID:   pt.SharedGroupID,
		PaidByMemberID:  pt.PaidByMemberID,
	}

	if msg := txIn.Validate(); msg != "" {
		writeErr(w, 422, "cannot create transaction: "+msg); return
	}

	tx, err := h.txRepo.Create(r.Context(), txIn)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}

	if err := h.repo.AdvanceNextDue(r.Context(), id); err != nil {
		writeErr(w, 500, "transaction created but failed to advance: "+err.Error()); return
	}

	writeJSON(w, 201, tx)
}