package handler

import (
	"net/http"

	"homebudget/internal/models"
	"homebudget/internal/repository"
)

type AnalyticsHandler struct {
	repo *repository.AnalyticsRepo
}

// Categories — GET /api/analytics/categories?from=&to=&type=expense&account_id=&category_id=&shared_group_id=
func (h *AnalyticsHandler) Categories(w http.ResponseWriter, r *http.Request) {
	f := models.AnalyticsFilter{
		DateFrom:      qStr(r, "from", ""),
		DateTo:        qStr(r, "to", ""),
		Type:          qStr(r, "type", "expense"),
		AccountID:     qInt64Ptr(r, "account_id"),
		CategoryID:    qInt64Ptr(r, "category_id"),
		SharedGroupID: qInt64Ptr(r, "shared_group_id"),
	}

	result, err := h.repo.CategoryBreakdown(r.Context(), f)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 200, result)
}

// Trends — GET /api/analytics/trends?from=&to=&granularity=month&account_id=&category_id=&shared_group_id=
func (h *AnalyticsHandler) Trends(w http.ResponseWriter, r *http.Request) {
	f := models.AnalyticsFilter{
		DateFrom:      qStr(r, "from", ""),
		DateTo:        qStr(r, "to", ""),
		Granularity:   qStr(r, "granularity", "month"),
		AccountID:     qInt64Ptr(r, "account_id"),
		CategoryID:    qInt64Ptr(r, "category_id"),
		SharedGroupID: qInt64Ptr(r, "shared_group_id"),
	}

	result, err := h.repo.Trends(r.Context(), f)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 200, result)
}