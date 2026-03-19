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

	accounts, err := h.accounts.ListWithBalances(ctx)
	log.Printf("  dashboard: accounts %v", time.Since(t0))
	if err != nil {
		writeErr(w, 500, "accounts: "+err.Error()); return
	}
	if accounts == nil {
		accounts = []models.AccountBalance{}
	}

	t1 := time.Now()
	summary, err := h.transactions.PeriodSummary(ctx, monthFrom, monthTo)
	log.Printf("  dashboard: summary %v", time.Since(t1))
	if err != nil {
		writeErr(w, 500, "summary: "+err.Error()); return
	}

	// settlements — простой список без вычислений
	t2 := time.Now()
	settlements, err := h.groups.ListSettlementSummariesFast(ctx)
	log.Printf("  dashboard: settlements %v", time.Since(t2))
	if err != nil {
		writeErr(w, 500, "settlements: "+err.Error()); return
	}

	t3 := time.Now()
	recent, err := h.transactions.Recent(ctx, 10)
	log.Printf("  dashboard: recent %v", time.Since(t3))
	if err != nil {
		writeErr(w, 500, "recent: "+err.Error()); return
	}
	if recent == nil {
		recent = []models.Transaction{}
	}

	t4 := time.Now()
	upcoming, err := h.planned.Upcoming(ctx, 14)
	log.Printf("  dashboard: upcoming %v", time.Since(t4))
	if err != nil {
		writeErr(w, 500, "upcoming: "+err.Error()); return
	}
	if upcoming == nil {
		upcoming = []models.PlannedTransaction{}
	}

	log.Printf("  dashboard: TOTAL %v", time.Since(t0))

	writeJSON(w, 200, models.Dashboard{
		Accounts:     accounts,
		CurrentMonth: *summary,
		Settlements:  settlements,
		Recent:       recent,
		Upcoming:     upcoming,
	})
}