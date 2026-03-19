package handler

import (
	"net/http"

	"homebudget/internal/models"
	"homebudget/internal/repository"
)

type TransactionHandler struct{ repo *repository.TransactionRepo }

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	f := models.TransactionFilter{
		DateFrom:       qStr(r, "from", ""),
		DateTo:         qStr(r, "to", ""),
		Search:         qStr(r, "search", ""),
		Type:           qStr(r, "type", ""),
		AccountID:      qInt64Ptr(r, "account_id"),
		CategoryID:     qInt64Ptr(r, "category_id"),
		SharedGroupID:  qInt64Ptr(r, "shared_group_id"),
		PaidByMemberID: qInt64Ptr(r, "paid_by_member_id"),
		IsShared:       qBoolPtr(r, "is_shared"),
		IsPending:      qBoolPtr(r, "is_pending"),
		Page:           qInt(r, "page", 1),
		Limit:          qInt(r, "limit", 20),
		SortBy:         qStr(r, "sort", "date"),
		SortDir:        qStr(r, "dir", "DESC"),
	}

	list, err := h.repo.List(r.Context(), f)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 200, list)
}

func (h *TransactionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	t, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if t == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, t)
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in models.CreateTransactionInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	t, err := h.repo.Create(r.Context(), in)
	if err != nil {
		if isFKError(err) {
			writeErr(w, 422, "referenced entity not found"); return
		}
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 201, t)
}

func (h *TransactionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	var in models.UpdateTransactionInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	t, err := h.repo.Update(r.Context(), id, in)
	if err != nil {
		if isFKError(err) {
			writeErr(w, 422, "referenced entity not found"); return
		}
		writeErr(w, 500, err.Error()); return
	}
	if t == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, t)
}

func (h *TransactionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeErr(w, 500, err.Error()); return
	}
	w.WriteHeader(204)
}

func (h *TransactionHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	t, err := h.repo.ConfirmPending(r.Context(), id)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if t == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, t)
}