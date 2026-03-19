package handler

import (
	"net/http"

	"homebudget/internal/models"
	"homebudget/internal/repository"
)

type SharedGroupHandler struct{ repo *repository.SharedGroupRepo }

func (h *SharedGroupHandler) List(w http.ResponseWriter, r *http.Request) {
	incl := r.URL.Query().Get("include_archived") == "true"
	items, err := h.repo.List(r.Context(), incl)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if items == nil {
		items = []models.SharedGroupWithMembers{}
	}
	writeJSON(w, 200, items)
}

func (h *SharedGroupHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	g, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if g == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, g)
}

func (h *SharedGroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in models.CreateSharedGroupInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	g, err := h.repo.Create(r.Context(), in)
	if err != nil {
		if isFKError(err) {
			writeErr(w, 422, "one or more member_ids not found"); return
		}
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 201, g)
}

func (h *SharedGroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	var in models.UpdateSharedGroupInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	g, err := h.repo.Update(r.Context(), id, in)
	if err != nil {
		if isFKError(err) {
			writeErr(w, 422, "one or more member_ids not found"); return
		}
		writeErr(w, 500, err.Error()); return
	}
	if g == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, g)
}

func (h *SharedGroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKError(err) {
			writeErr(w, 409, "group has transactions, archive instead"); return
		}
		writeErr(w, 500, err.Error()); return
	}
	w.WriteHeader(204)
}

// Settlement — GET /api/groups/{id}/settlement?from=&to=
func (h *SharedGroupHandler) Settlement(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	s, err := h.repo.GetSettlement(r.Context(), id,
		qStr(r, "from", ""), qStr(r, "to", ""))
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if s == nil {
		writeErr(w, 404, "group not found"); return
	}
	writeJSON(w, 200, s)
}

// Turnover — GET /api/groups/{id}/turnover?from=&to=
func (h *SharedGroupHandler) Turnover(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	t, err := h.repo.GetTurnover(r.Context(), id,
		qStr(r, "from", ""), qStr(r, "to", ""))
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if t == nil {
		writeErr(w, 404, "group not found"); return
	}
	writeJSON(w, 200, t)
}