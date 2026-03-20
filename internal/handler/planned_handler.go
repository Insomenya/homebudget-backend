package handler

import (
	"net/http"

	"homebudget/internal/models"
	"homebudget/internal/repository"
)

type PlannedHandler struct {
	repo   *repository.PlannedRepo
	txRepo *repository.TransactionRepo
}

func (h *PlannedHandler) List(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("all") != "true"
	items, err := h.repo.List(r.Context(), activeOnly)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 200, items)
}

func (h *PlannedHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	p, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if p == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, p)
}

func (h *PlannedHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in models.CreatePlannedInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	p, err := h.repo.Create(r.Context(), in)
	if err != nil {
		if isFKError(err) {
			writeErr(w, 422, "referenced entity not found"); return
		}
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 201, p)
}

func (h *PlannedHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	var in models.UpdatePlannedInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, 400, err.Error()); return
	}
	if msg := in.Validate(); msg != "" {
		writeErr(w, 422, msg); return
	}
	p, err := h.repo.Update(r.Context(), id, in)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if p == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, p)
}

func (h *PlannedHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeErr(w, 500, err.Error()); return
	}
	w.WriteHeader(204)
}

func (h *PlannedHandler) Upcoming(w http.ResponseWriter, r *http.Request) {
	days := qInt(r, "days", 30)
	items, err := h.repo.Upcoming(r.Context(), days)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	writeJSON(w, 200, items)
}

func (h *PlannedHandler) Materialize(w http.ResponseWriter, r *http.Request) {
	created := h.repo.MaterializeReminders(r.Context())
	writeJSON(w, 200, map[string]int{"created": created})
}

func (h *PlannedHandler) Activate(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	p, err := h.repo.ActivatePlanned(r.Context(), id)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if p == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, p)
}

// Reminders
func (h *PlannedHandler) ListReminders(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListActiveReminders(r.Context())
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if items == nil {
		items = []models.PlannedReminder{}
	}
	writeJSON(w, 200, items)
}

func (h *PlannedHandler) ExecuteReminder(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	var in models.ExecuteReminderInput
	if err := readJSON(r, &in); err != nil {
		// Allow empty body — will use defaults
		in = models.ExecuteReminderInput{}
	}
	tx, err := h.repo.ExecuteReminder(r.Context(), id, in, h.txRepo)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if tx == nil {
		writeErr(w, 404, "not found"); return
	}
	writeJSON(w, 200, tx)
}

func (h *PlannedHandler) UndoReminder(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r)
	if err != nil {
		writeErr(w, 400, "invalid id"); return
	}
	if err := h.repo.UndoReminder(r.Context(), id, h.txRepo); err != nil {
		writeErr(w, 500, err.Error()); return
	}
	w.WriteHeader(204)
}

func (h *PlannedHandler) Forecast(w http.ResponseWriter, r *http.Request) {
	days := qInt(r, "days", 30)
	items, err := h.repo.Forecast(r.Context(), days)
	if err != nil {
		writeErr(w, 500, err.Error()); return
	}
	if items == nil {
		items = []models.PlannedForecastItem{}
	}
	writeJSON(w, 200, items)
}