package handler

import (
	"net/http"

	"homebudget/internal/models"
	"homebudget/internal/repository"
)

type AccountHandler struct{ repo *repository.AccountRepo }

func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	incl := r.URL.Query().Get("include_archived") == "true"
	items, err := h.repo.List(r.Context(), incl)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if items == nil {
		items = []models.Account{}
	}
	writeJSON(w, 200, items)
}

func (h *AccountHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	a, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if a == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, a)
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in models.CreateAccountInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	a, err := h.repo.Create(r.Context(), in)
	if err != nil {
		if isFKError(err) {
			writeErr(w, 422, "member_id not found"); return
		}
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 201, a)
}

func (h *AccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	var in models.UpdateAccountInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	a, err := h.repo.Update(r.Context(), id, in)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if a == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, a)
}

func (h *AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKError(err) {
			writeErr(w, 409, "account has transactions, archive instead"); return
		}
		writeErr(w, 500, err.Error()); return
	}
	w.WriteHeader(204)
}