// FILE: internal/repository/planned_repo.go
package repository

import (
	"context"
	"database/sql"
	"log"
	"time"

	"homebudget/internal/models"
)

type PlannedRepo struct{ db *sql.DB }

func NewPlannedRepo(db *sql.DB) *PlannedRepo { return &PlannedRepo{db: db} }

const ptCols = `id, name, amount, type, category_id,
	shared_group_id, paid_by_member_id, loan_id, recurrence,
	start_date, end_date, next_due, original_day,
	notify_days_before, overdue_days_limit,
	is_active, created_at, updated_at`

func scanPT(s scannable) (models.PlannedTransaction, error) {
	var p models.PlannedTransaction
	var catID, grpID, paidID, loanID sql.NullInt64
	var endDate sql.NullString
	var isActive int

	err := s.Scan(
		&p.ID, &p.Name, &p.Amount, &p.Type,
		&catID, &grpID, &paidID, &loanID,
		&p.Recurrence, &p.StartDate, &endDate, &p.NextDue,
		&p.OriginalDay, &p.NotifyDaysBefore, &p.OverdueDaysLimit,
		&isActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if catID.Valid {
		p.CategoryID = &catID.Int64
	}
	if grpID.Valid {
		p.SharedGroupID = &grpID.Int64
	}
	if paidID.Valid {
		p.PaidByMemberID = &paidID.Int64
	}
	if loanID.Valid {
		p.LoanID = &loanID.Int64
	}
	if endDate.Valid {
		p.EndDate = &endDate.String
	}
	p.IsActive = isActive == 1
	return p, err
}

func (r *PlannedRepo) List(ctx context.Context, activeOnly bool) ([]models.PlannedTransaction, error) {
	q := "SELECT " + ptCols + " FROM planned_transactions"
	if activeOnly {
		q += " WHERE is_active=1"
	}
	q += " ORDER BY next_due ASC"

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.PlannedTransaction, 0)
	for rows.Next() {
		p, err := scanPT(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PlannedRepo) GetByID(ctx context.Context, id int64) (*models.PlannedTransaction, error) {
	p, err := scanPT(r.db.QueryRowContext(ctx,
		"SELECT "+ptCols+" FROM planned_transactions WHERE id=?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlannedRepo) Create(ctx context.Context, in models.CreatePlannedInput) (*models.PlannedTransaction, error) {
	now := ts()
	originalDay := models.ExtractDay(in.StartDate)

	res, err := r.db.ExecContext(ctx,
		`INSERT INTO planned_transactions
		 (name,amount,type,category_id,shared_group_id,paid_by_member_id,loan_id,
		  recurrence,start_date,end_date,next_due,original_day,
		  notify_days_before,overdue_days_limit,is_active,
		  created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,1,?,?)`,
		in.Name, in.Amount, in.Type, in.CategoryID,
		in.SharedGroupID, in.PaidByMemberID, in.LoanID,
		in.Recurrence, in.StartDate, in.EndDate, in.StartDate,
		originalDay, in.NotifyDaysBefore, in.OverdueDaysLimit,
		now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *PlannedRepo) Update(ctx context.Context, id int64, in models.UpdatePlannedInput) (*models.PlannedTransaction, error) {
	now := ts()
	originalDay := models.ExtractDay(in.StartDate)

	_, err := r.db.ExecContext(ctx,
		`UPDATE planned_transactions
		 SET name=?,amount=?,type=?,category_id=?,
		     shared_group_id=?,paid_by_member_id=?,loan_id=?,
		     recurrence=?,start_date=?,end_date=?,next_due=?,
		     original_day=?,notify_days_before=?,overdue_days_limit=?,
		     updated_at=?
		 WHERE id=?`,
		in.Name, in.Amount, in.Type, in.CategoryID,
		in.SharedGroupID, in.PaidByMemberID, in.LoanID,
		in.Recurrence, in.StartDate, in.EndDate, in.StartDate,
		originalDay, in.NotifyDaysBefore, in.OverdueDaysLimit,
		now, id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *PlannedRepo) UpdateAmount(ctx context.Context, id int64, amount float64) error {
	now := ts()
	_, err := r.db.ExecContext(ctx,
		"UPDATE planned_transactions SET amount=?, updated_at=? WHERE id=?",
		amount, now, id)
	return err
}

func (r *PlannedRepo) Delete(ctx context.Context, id int64) error {
	// Reminders cascade-deleted by FK
	_, err := r.db.ExecContext(ctx, "DELETE FROM planned_transactions WHERE id=?", id)
	return err
}

func (r *PlannedRepo) Upcoming(ctx context.Context, days int) ([]models.PlannedTransaction, error) {
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, days).Format("2006-01-02")

	rows, err := r.db.QueryContext(ctx,
		"SELECT "+ptCols+` FROM planned_transactions
		 WHERE is_active=1 AND next_due <= ?
		 ORDER BY next_due ASC`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.PlannedTransaction, 0)
	for rows.Next() {
		p, err := scanPT(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ── Reminders ───────────────────────────────────────

const reminderCols = `id, planned_id, due_date, amount, transaction_id, prev_next_due, is_executed, created_at`

func scanReminder(s scannable) (models.PlannedReminder, error) {
	var r models.PlannedReminder
	var txID sql.NullInt64
	var executed int
	err := s.Scan(&r.ID, &r.PlannedID, &r.DueDate, &r.Amount, &txID, &r.PrevNextDue, &executed, &r.CreatedAt)
	if txID.Valid {
		r.TransactionID = &txID.Int64
	}
	r.IsExecuted = executed == 1
	return r, err
}

func (r *PlannedRepo) GetReminderByID(ctx context.Context, id int64) (*models.PlannedReminder, error) {
	rem, err := scanReminder(r.db.QueryRowContext(ctx,
		"SELECT "+reminderCols+" FROM planned_reminders WHERE id=?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rem, nil
}

func (r *PlannedRepo) ListActiveReminders(ctx context.Context) ([]models.PlannedReminder, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+reminderCols+" FROM planned_reminders WHERE is_executed=0 ORDER BY due_date ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.PlannedReminder, 0)
	for rows.Next() {
		rem, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rem)
	}
	return out, rows.Err()
}

func (r *PlannedRepo) ListRemindersForPlanned(ctx context.Context, plannedID int64) ([]models.PlannedReminder, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+reminderCols+" FROM planned_reminders WHERE planned_id=? ORDER BY due_date ASC", plannedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.PlannedReminder, 0)
	for rows.Next() {
		rem, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rem)
	}
	return out, rows.Err()
}

func (r *PlannedRepo) CreateReminder(ctx context.Context, plannedID int64, dueDate string, amount float64, prevNextDue string) (*models.PlannedReminder, error) {
	now := ts()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO planned_reminders (planned_id, due_date, amount, prev_next_due, is_executed, created_at)
		 VALUES (?,?,?,?,0,?)`,
		plannedID, dueDate, amount, prevNextDue, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetReminderByID(ctx, id)
}

func (r *PlannedRepo) DeleteReminder(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM planned_reminders WHERE id=?", id)
	return err
}

func (r *PlannedRepo) DeleteUnexecutedRemindersForPlanned(ctx context.Context, plannedID int64) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM planned_reminders WHERE planned_id=? AND is_executed=0", plannedID)
	return err
}

// ExecuteReminder создаёт транзакцию и помечает reminder как executed.
func (r *PlannedRepo) ExecuteReminder(ctx context.Context, reminderID int64, in models.ExecuteReminderInput, txRepo *TransactionRepo) (*models.Transaction, error) {
	rem, err := r.GetReminderByID(ctx, reminderID)
	if err != nil || rem == nil {
		return nil, err
	}
	if rem.IsExecuted {
		// Already executed — return existing transaction
		if rem.TransactionID != nil {
			return txRepo.GetByID(ctx, *rem.TransactionID)
		}
		return nil, nil
	}

	pt, err := r.GetByID(ctx, rem.PlannedID)
	if err != nil {
		return nil, err
	}
	if pt == nil {
		return nil, nil
	}

	amount := in.Amount
	if amount <= 0 {
		amount = rem.Amount
	}
	date := in.Date
	if date == "" {
		date = rem.DueDate
	}

	txIn := models.CreateTransactionInput{
		Date:           date,
		Amount:         amount,
		Description:    pt.Name,
		Type:           pt.Type,
		AccountID:      in.AccountID,
		CategoryID:     pt.CategoryID,
		SharedGroupID:  pt.SharedGroupID,
		PaidByMemberID: pt.PaidByMemberID,
		LoanID:         pt.LoanID,
		ReminderID:     &rem.ID,
	}

	tx, err := txRepo.Create(ctx, txIn)
	if err != nil {
		return nil, err
	}

	// Mark reminder as executed
	now := ts()
	r.db.ExecContext(ctx,
		"UPDATE planned_reminders SET is_executed=1, transaction_id=?, created_at=? WHERE id=?",
		tx.ID, now, rem.ID)

	// Advance planned next_due
	r.advanceNextDue(ctx, pt)

	return tx, nil
}

// UndoReminder отменяет проводку: удаляет транзакцию, возвращает напоминание в active,
// возвращает next_due к prev_next_due.
func (r *PlannedRepo) UndoReminder(ctx context.Context, reminderID int64, txRepo *TransactionRepo) error {
	rem, err := r.GetReminderByID(ctx, reminderID)
	if err != nil || rem == nil {
		return err
	}
	if !rem.IsExecuted {
		return nil
	}

	// Delete the transaction
	if rem.TransactionID != nil {
		txRepo.Delete(ctx, *rem.TransactionID)
	}

	// Check if planned still exists
	pt, _ := r.GetByID(ctx, rem.PlannedID)
	if pt != nil {
		// Restore previous next_due
		now := ts()
		r.db.ExecContext(ctx,
			"UPDATE planned_transactions SET next_due=?, is_active=1, updated_at=? WHERE id=?",
			rem.PrevNextDue, now, rem.PlannedID)
	}

	// Mark reminder as not executed, clear transaction_id
	r.db.ExecContext(ctx,
		"UPDATE planned_reminders SET is_executed=0, transaction_id=NULL WHERE id=?",
		rem.ID)

	return nil
}

// UndoReminderByTxID — отмена проводки по ID транзакции (для кнопки "отменить" в таблице транзакций).
func (r *PlannedRepo) UndoReminderByTxID(ctx context.Context, txID int64, txRepo *TransactionRepo) error {
	var remID int64
	err := r.db.QueryRowContext(ctx,
		"SELECT id FROM planned_reminders WHERE transaction_id=?", txID).Scan(&remID)
	if err == sql.ErrNoRows {
		return nil // No reminder linked
	}
	if err != nil {
		return err
	}
	return r.UndoReminder(ctx, remID, txRepo)
}

func (r *PlannedRepo) advanceNextDue(ctx context.Context, pt *models.PlannedTransaction) {
	next, active := models.AdvanceDate(pt.NextDue, pt.Recurrence, pt.EndDate, pt.OriginalDay)
	now := ts()
	r.db.ExecContext(ctx,
		"UPDATE planned_transactions SET next_due=?, is_active=?, updated_at=? WHERE id=?",
		next, boolInt(active), now, pt.ID)
}

// MaterializeReminders создаёт напоминания для платежей, у которых
// next_due попадает в окно [today - overdue, today + notify_days_before].
// Также деактивирует платежи, которые просрочены больше overdue_days_limit.
func (r *PlannedRepo) MaterializeReminders(ctx context.Context) int {
	active, err := r.List(ctx, true)
	if err != nil {
		return 0
	}

	created := 0

	for _, pt := range active {
		if !pt.IsActive {
			continue
		}

		notifyCutoff := time.Now().AddDate(0, 0, pt.NotifyDaysBefore).Format("2006-01-02")

		// Check if next_due is within notification window
		if pt.NextDue > notifyCutoff {
			continue
		}

		// Check if overdue beyond limit
		overdueCutoff := time.Now().AddDate(0, 0, -pt.OverdueDaysLimit).Format("2006-01-02")
		if pt.NextDue < overdueCutoff {
			// Deactivate
			now := ts()
			r.db.ExecContext(ctx,
				"UPDATE planned_transactions SET is_active=0, updated_at=? WHERE id=?",
				now, pt.ID)
			log.Printf("  deactivated planned %d (overdue > %d days)", pt.ID, pt.OverdueDaysLimit)
			continue
		}

		// Check if reminder already exists for this due date
		var cnt int
		r.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM planned_reminders WHERE planned_id=? AND due_date=?",
			pt.ID, pt.NextDue).Scan(&cnt)
		if cnt > 0 {
			continue
		}

		_, err := r.CreateReminder(ctx, pt.ID, pt.NextDue, pt.Amount, pt.NextDue)
		if err != nil {
			log.Printf("  reminder create error for planned %d: %v", pt.ID, err)
			continue
		}
		created++
	}

	return created
}

// Forecast — прогноз влияния отложенных платежей на балансы.
func (r *PlannedRepo) Forecast(ctx context.Context, days int) ([]models.PlannedForecastItem, error) {
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, days).Format("2006-01-02")

	active, err := r.List(ctx, true)
	if err != nil {
		return nil, err
	}

	items := make([]models.PlannedForecastItem, 0)
	for _, pt := range active {
		if !pt.IsActive {
			continue
		}

		// Collect all due dates within range
		currentDue := pt.NextDue
		for currentDue <= cutoff {
			items = append(items, models.PlannedForecastItem{
				PlannedID: pt.ID,
				Name:      pt.Name,
				Amount:    pt.Amount,
				Type:      pt.Type,
				DueDate:   currentDue,
				Enabled:   true,
			})

			if pt.Recurrence == models.RecurrenceOnce {
				break
			}
			next, stillActive := models.AdvanceDate(currentDue, pt.Recurrence, pt.EndDate, pt.OriginalDay)
			if !stillActive || next == currentDue {
				break
			}
			currentDue = next
		}
	}

	return items, nil
}

// ActivatePlanned — активирует платёж, назначая следующую корректную дату в будущем.
func (r *PlannedRepo) ActivatePlanned(ctx context.Context, id int64) (*models.PlannedTransaction, error) {
	pt, err := r.GetByID(ctx, id)
	if err != nil || pt == nil {
		return nil, err
	}

	todayStr := time.Now().Format("2006-01-02")
	nextDue := pt.NextDue

	// Advance until next_due is in the future
	for nextDue <= todayStr {
		if pt.Recurrence == models.RecurrenceOnce {
			break
		}
		next, _ := models.AdvanceDate(nextDue, pt.Recurrence, pt.EndDate, pt.OriginalDay)
		if next == nextDue {
			break
		}
		nextDue = next
	}

	now := ts()
	_, err = r.db.ExecContext(ctx,
		"UPDATE planned_transactions SET next_due=?, is_active=1, updated_at=? WHERE id=?",
		nextDue, now, id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}