package handler

import (
	"net/http"

	"homebudget/internal/models"
	"homebudget/internal/repository"
)

type CategoryHandler struct{ repo *repository.CategoryRepo }

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	incl := r.URL.Query().Get("include_archived") == "true"
	items, err := h.repo.List(r.Context(), incl)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if items == nil {
		items = []models.Category{}
	}
	writeJSON(w, 200, items)
}

func (h *CategoryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	c, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if c == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, c)
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in models.CreateCategoryInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	c, err := h.repo.Create(r.Context(), in)
	if err != nil {
		if isFKError(err) {
			writeErr(w, 422, "parent_id not found"); return
		}
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 201, c)
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	var in models.UpdateCategoryInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	c, err := h.repo.Update(r.Context(), id, in)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if c == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, c)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKError(err) {
			writeErr(w, 409, "category has linked data, archive instead"); return
		}
		writeErr(w, 500, err.Error()); return
	}
	w.WriteHeader(204)
}