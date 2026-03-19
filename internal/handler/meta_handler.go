package handler

import (
	"net/http"
	"strings"

	"homebudget/internal/models"
	"homebudget/internal/repository"
)

type MetaHandler struct {
	repo *repository.LookupRepo
}

func (h *MetaHandler) GetMeta(w http.ResponseWriter, r *http.Request) {
	meta, err := h.repo.GetMeta(r.Context())
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 200, meta)
}

func (h *MetaHandler) ListGroup(w http.ResponseWriter, r *http.Request) {
	group := qStr(r, "group", "")
	if group == "" {
		items, err := h.repo.ListAll(r.Context())
		if err != nil {
			writeErr(w, 500, err.Error()); return
		}
		writeJSON(w, 200, items)
		return
	}

	items, err := h.repo.ListByGroup(r.Context(), group, false)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 200, items)
}

func (h *MetaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in models.CreateLookupInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	v, err := h.repo.Create(r.Context(), in)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeErr(w, 409, "value already exists in this group"); return
		}
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 201, v)
}

func (h *MetaHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	var in models.UpdateLookupInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	v, err := h.repo.Update(r.Context(), id, in)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if v == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, v)
}

func (h *MetaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeErr(w, 500, err.Error()); return
	}
	w.WriteHeader(204)
}