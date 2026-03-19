package handler

import (
	"net/http"

	"homebudget/internal/models"
	"homebudget/internal/repository"
)

type BudgetHandler struct {
	repo *repository.BudgetRepo
}

func (h *BudgetHandler) GetTable(w http.ResponseWriter, r *http.Request) {
	page := qInt(r, "page", 1)
	limit := qInt(r, "limit", 30)
	t, err := h.repo.GetTable(r.Context(), page, limit)
	if err != nil { writeErr(w, 500, err.Error()); return }
	writeJSON(w, 200, t)
}

func (h *BudgetHandler) CreateColumn(w http.ResponseWriter, r *http.Request) {
	var in models.CreateBudgetColumnInput
	if err := readJSON(r, &in); err != nil { writeErr(w, 400, err.Error()); return }
	c, err := h.repo.CreateColumn(r.Context(), in)
	if err != nil { writeErr(w, 500, err.Error()); return }
	writeJSON(w, 201, c)
}

func (h *BudgetHandler) DeleteColumn(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil { writeErr(w, 400, "invalid id"); return }
	if err := h.repo.DeleteColumn(r.Context(), id); err != nil { writeErr(w, 500, err.Error()); return }
	w.WriteHeader(204)
}

func (h *BudgetHandler) CreateRow(w http.ResponseWriter, r *http.Request) {
	var in models.CreateBudgetRowInput
	if err := readJSON(r, &in); err != nil { writeErr(w, 400, err.Error()); return }
	row, err := h.repo.CreateRow(r.Context(), in)
	if err != nil { writeErr(w, 500, err.Error()); return }
	writeJSON(w, 201, row)
}

func (h *BudgetHandler) DeleteRow(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil { writeErr(w, 400, "invalid id"); return }
	if err := h.repo.DeleteRow(r.Context(), id); err != nil { writeErr(w, 500, err.Error()); return }
	w.WriteHeader(204)
}

func (h *BudgetHandler) UpdateCell(w http.ResponseWriter, r *http.Request) {
	var in models.UpdateBudgetCellInput
	if err := readJSON(r, &in); err != nil { writeErr(w, 400, err.Error()); return }
	if err := h.repo.UpdateCell(r.Context(), in); err != nil { writeErr(w, 500, err.Error()); return }
	w.WriteHeader(204)
}

func (h *BudgetHandler) ToggleExecuted(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil { writeErr(w, 400, "invalid id"); return }
	if err := h.repo.ToggleExecuted(r.Context(), id); err != nil { writeErr(w, 500, err.Error()); return }
	w.WriteHeader(204)
}