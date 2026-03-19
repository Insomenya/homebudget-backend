package handler

import (
	"net/http"

	"homebudget/internal/models"
	"homebudget/internal/repository"
)

type LoanHandler struct {
	repo *repository.LoanRepo
}

func (h *LoanHandler) List(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("all") != "true"
	items, err := h.repo.List(r.Context(), activeOnly)
	if err != nil { writeErr(w, 500, err.Error()); return }
	writeJSON(w, 200, items)
}

func (h *LoanHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil { writeErr(w, 400, "invalid id"); return }
	l, err := h.repo.GetByID(r.Context(), id)
	if err != nil { writeErr(w, 500, err.Error()); return }
	if l == nil { writeErr(w, 404, "not found"); return }
	writeJSON(w, 200, l)
}

func (h *LoanHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in models.CreateLoanInput
	if err := readJSON(r, &in); err != nil { writeErr(w, 400, err.Error()); return }
	if msg := in.Validate(); msg != "" { writeErr(w, 422, msg); return }
	l, err := h.repo.Create(r.Context(), in)
	if err != nil { writeErr(w, 500, err.Error()); return }
	writeJSON(w, 201, l)
}

func (h *LoanHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil { writeErr(w, 400, "invalid id"); return }
	var in models.UpdateLoanInput
	if err := readJSON(r, &in); err != nil { writeErr(w, 400, err.Error()); return }
	l, err := h.repo.Update(r.Context(), id, in)
	if err != nil { writeErr(w, 500, err.Error()); return }
	if l == nil { writeErr(w, 404, "not found"); return }
	writeJSON(w, 200, l)
}

func (h *LoanHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil { writeErr(w, 400, "invalid id"); return }
	if err := h.repo.Delete(r.Context(), id); err != nil { writeErr(w, 500, err.Error()); return }
	w.WriteHeader(204)
}

func (h *LoanHandler) DailySchedule(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil { writeErr(w, 400, "invalid id"); return }
	from := qStr(r, "from", "")
	to := qStr(r, "to", "")
	if from == "" || to == "" { writeErr(w, 422, "from and to required"); return }
	s, err := h.repo.GetDailySchedule(r.Context(), id, from, to)
	if err != nil { writeErr(w, 500, err.Error()); return }
	if s == nil { writeErr(w, 404, "loan not found"); return }
	writeJSON(w, 200, s)
}