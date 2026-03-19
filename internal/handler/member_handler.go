package handler

import (
	"net/http"

	"homebudget/internal/models"
	"homebudget/internal/repository"
)

type MemberHandler struct{ repo *repository.MemberRepo }

func (h *MemberHandler) List(w http.ResponseWriter, r *http.Request) {
	incl := r.URL.Query().Get("include_archived") == "true"
	items, err := h.repo.List(r.Context(), incl)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if items == nil {
		items = []models.Member{}
	}
	writeJSON(w, 200, items)
}

func (h *MemberHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	m, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if m == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, m)
}

func (h *MemberHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in models.CreateMemberInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	m, err := h.repo.Create(r.Context(), in)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 201, m)
}

func (h *MemberHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	var in models.UpdateMemberInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	m, err := h.repo.Update(r.Context(), id, in)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if m == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, m)
}

func (h *MemberHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKError(err) {
			writeErr(w, 409, "member has linked data, archive instead"); return
		}
		writeErr(w, 500, err.Error()); return
	}
	w.WriteHeader(204)
}