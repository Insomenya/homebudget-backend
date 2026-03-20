package handler

import (
	"log"
	"net/http"
	"time"

	"homebudget/internal/models"
	"homebudget/internal/repository"
)

type DashboardHandler struct {
	accounts     *repository.AccountRepo
	transactions *repository.TransactionRepo
	groups       *repository.SharedGroupRepo
	planned      *repository.PlannedRepo
}

func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	now := time.Now()
	monthFrom := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	monthTo := now.Format("2006-01-02")

	t0 := time.Now()

	// Auto-materialize reminders
	h.planned.MaterializeReminders(ctx)

	accounts, err := h.accounts.ListWithBalances(ctx)
	if err != nil {
		writeErr(w, 500, "accounts: "+err.Error()); return
	}
	if accounts == nil {
		accounts = []models.AccountBalance{}
	}

	summary, err := h.transactions.PeriodSummary(ctx, monthFrom, monthTo)
	if err != nil {
		writeErr(w, 500, "summary: "+err.Error()); return
	}

	settlements, err := h.groups.ListSettlementSummariesFast(ctx)
	if err != nil {
		writeErr(w, 500, "settlements: "+err.Error()); return
	}

	recent, err := h.transactions.Recent(ctx, 10)
	if err != nil {
		writeErr(w, 500, "recent: "+err.Error()); return
	}
	if recent == nil {
		recent = []models.Transaction{}
	}

	upcoming, err := h.planned.Upcoming(ctx, 14)
	if err != nil {
		writeErr(w, 500, "upcoming: "+err.Error()); return
	}
	if upcoming == nil {
		upcoming = []models.PlannedTransaction{}
	}

	reminders, err := h.planned.ListActiveReminders(ctx)
	if err != nil {
		writeErr(w, 500, "reminders: "+err.Error()); return
	}
	if reminders == nil {
		reminders = []models.PlannedReminder{}
	}

	log.Printf("  dashboard: TOTAL %v", time.Since(t0))

	writeJSON(w, 200, models.Dashboard{
		Accounts:     accounts,
		CurrentMonth: *summary,
		Settlements:  settlements,
		Recent:       recent,
		Upcoming:     upcoming,
		Reminders:    reminders,
	})
}