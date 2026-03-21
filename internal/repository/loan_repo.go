// FILE: internal/repository/loan_repo.go
package repository

import (
	"context"
	"database/sql"

	"homebudget/internal/models"
)

type LoanRepo struct{ db *sql.DB }

func NewLoanRepo(db *sql.DB) *LoanRepo { return &LoanRepo{db: db} }

const loanCols = `id, name, principal, annual_rate, start_date, end_date,
	monthly_payment, already_paid, account_id, default_account_id, loan_account_id,
	category_id, planned_id, accounting_start_date, initial_accrued_interest, is_active, created_at, updated_at`

func scanLoan(s scannable) (models.Loan, error) {
	var l models.Loan
	var accID, defAccID, loanAccID, catID, plannedID sql.NullInt64
	var active int
	err := s.Scan(&l.ID, &l.Name, &l.Principal, &l.AnnualRate,
		&l.StartDate, &l.EndDate, &l.MonthlyPayment, &l.AlreadyPaid,
		&accID, &defAccID, &loanAccID, &catID, &plannedID, &l.AccountingStartDate, &l.InitialAccruedInterest, &active,
		&l.CreatedAt, &l.UpdatedAt)
	if accID.Valid {
		l.AccountID = &accID.Int64
	}
	if defAccID.Valid {
		l.DefaultAccountID = &defAccID.Int64
	}
	if loanAccID.Valid {
		l.LoanAccountID = &loanAccID.Int64
	}
	if catID.Valid {
		l.CategoryID = &catID.Int64
	}
	if plannedID.Valid {
		l.PlannedID = &plannedID.Int64
	}
	l.IsActive = active == 1
	return l, err
}

func (r *LoanRepo) List(ctx context.Context, activeOnly bool) ([]models.Loan, error) {
	q := "SELECT " + loanCols + " FROM loans"
	if activeOnly {
		q += " WHERE is_active=1"
	}
	q += " ORDER BY start_date DESC"

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Loan, 0)
	for rows.Next() {
		l, err := scanLoan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *LoanRepo) GetByID(ctx context.Context, id int64) (*models.Loan, error) {
	l, err := scanLoan(r.db.QueryRowContext(ctx, "SELECT "+loanCols+" FROM loans WHERE id=?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *LoanRepo) Create(ctx context.Context, in models.CreateLoanInput) (*models.Loan, error) {
	now := ts()
	accountingStart := in.StartDate
	if in.AccountingStartDate != nil && *in.AccountingStartDate != "" {
		accountingStart = *in.AccountingStartDate
	}
	pmt := models.CalcMonthlyPaymentForLoan(in.Principal, in.AlreadyPaid, in.AnnualRate, in.StartDate, in.EndDate)
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO loans (name,principal,annual_rate,start_date,end_date,monthly_payment,already_paid,
		 account_id,default_account_id,loan_account_id,category_id,planned_id,
		 accounting_start_date,initial_accrued_interest,is_active,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,NULL,?,NULL,?,?,1,?,?)`,
		in.Name, in.Principal, in.AnnualRate, in.StartDate, in.EndDate, pmt,
		in.AlreadyPaid, in.AccountID, in.DefaultAccountID,
		in.CategoryID, accountingStart, in.InitialAccruedInterest, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *LoanRepo) Update(ctx context.Context, id int64, in models.UpdateLoanInput) (*models.Loan, error) {
	now := ts()
	_, err := r.db.ExecContext(ctx,
		`UPDATE loans SET name=?, annual_rate=?, default_account_id=?, category_id=?, is_active=?, updated_at=? WHERE id=?`,
		in.Name, in.AnnualRate, in.DefaultAccountID, in.CategoryID, boolInt(in.IsActive), now, id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *LoanRepo) SetLoanAccountID(ctx context.Context, loanID, accountID int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE loans SET loan_account_id=? WHERE id=?", accountID, loanID)
	return err
}

func (r *LoanRepo) SetPlannedID(ctx context.Context, loanID, plannedID int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE loans SET planned_id=? WHERE id=?", plannedID, loanID)
	return err
}

func (r *LoanRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM loans WHERE id=?", id)
	return err
}

// DeleteWithCleanup удаляет кредит вместе с:
// - отложенным платежом (и его неисполненными напоминаниями)
// - скрытым loan_account (если нет транзакций, иначе оставляем)
// Проведённые транзакции НЕ удаляются.
func (r *LoanRepo) DeleteWithCleanup(ctx context.Context, id int64, planned *PlannedRepo, accounts *AccountRepo) error {
	loan, err := r.GetByID(ctx, id)
	if err != nil || loan == nil {
		return err
	}

	// Delete planned transaction (cascade deletes reminders)
	if loan.PlannedID != nil {
		planned.Delete(ctx, *loan.PlannedID)
	}

	// Delete loan account if no transactions reference it
	if loan.LoanAccountID != nil {
		var cnt int
		r.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM transactions WHERE account_id=? OR to_account_id=?",
			*loan.LoanAccountID, *loan.LoanAccountID).Scan(&cnt)
		if cnt == 0 {
			accounts.Delete(ctx, *loan.LoanAccountID)
		}
	}

	// Unlink loan_id from remaining transactions (don't delete them)
	r.db.ExecContext(ctx, "UPDATE transactions SET loan_id=NULL WHERE loan_id=?", id)

	return r.Delete(ctx, id)
}

// RecalcPayment пересчитывает monthly_payment на основе текущего тела долга.
// Вычисляет оставшийся долг, прогоняя день за днём с учётом всех платежей.
func (r *LoanRepo) RecalcPayment(ctx context.Context, id int64, planned *PlannedRepo) error {
	loan, err := r.GetByID(ctx, id)
	if err != nil || loan == nil {
		return err
	}

	// Get all payments for this loan
	rows, err := r.db.QueryContext(ctx, txBase+" WHERE t.loan_id = ? ORDER BY t.date", id)
	if err != nil {
		return err
	}
	defer rows.Close()

	var payments []models.Transaction
	for rows.Next() {
		t, err := scanTx(rows)
		if err != nil {
			return err
		}
		payments = append(payments, t)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Calculate current debt by replaying all days
	today := models.TodayStr()
	schedule := models.BuildDailySchedule(*loan, payments, loan.AccountingStartDate, today)

	remainingPrincipal := schedule.CurrentDebt
	if remainingPrincipal <= 0 {
		// Loan is paid off
		now := ts()
		r.db.ExecContext(ctx,
			"UPDATE loans SET monthly_payment=0, updated_at=? WHERE id=?",
			now, id)
		if loan.PlannedID != nil {
			planned.UpdateAmount(ctx, *loan.PlannedID, 0)
		}
		return nil
	}

	// Remaining months from today to end_date
	remainingMonths := models.CalcTermMonths(today, loan.EndDate)
	if remainingMonths < 1 {
		remainingMonths = 1
	}

	newPayment := models.CalcMonthlyPayment(remainingPrincipal, loan.AnnualRate, remainingMonths)

	now := ts()
	r.db.ExecContext(ctx,
		"UPDATE loans SET monthly_payment=?, updated_at=? WHERE id=?",
		newPayment, now, id)

	// Update planned amount
	if loan.PlannedID != nil {
		planned.UpdateAmount(ctx, *loan.PlannedID, newPayment)
	}

	return nil
}

func (r *LoanRepo) GetDailySchedule(ctx context.Context, id int64, from, to string) (*models.LoanDailySchedule, error) {
	loan, err := r.GetByID(ctx, id)
	if err != nil || loan == nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, txBase+" WHERE t.loan_id = ? ORDER BY t.date", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []models.Transaction
	for rows.Next() {
		t, err := scanTx(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return models.BuildDailySchedule(*loan, payments, from, to), nil
}