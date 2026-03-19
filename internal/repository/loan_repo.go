package repository

import (
	"context"
	"database/sql"

	"homebudget/internal/models"
)

type LoanRepo struct{ db *sql.DB }

func NewLoanRepo(db *sql.DB) *LoanRepo { return &LoanRepo{db: db} }

const loanCols = `id, name, principal, annual_rate, term_months, start_date,
	monthly_payment, already_paid, account_id, category_id, is_active, created_at, updated_at`

func scanLoan(s scannable) (models.Loan, error) {
	var l models.Loan
	var accID, catID sql.NullInt64
	var active int
	err := s.Scan(&l.ID, &l.Name, &l.Principal, &l.AnnualRate, &l.TermMonths,
		&l.StartDate, &l.MonthlyPayment, &l.AlreadyPaid,
		&accID, &catID, &active, &l.CreatedAt, &l.UpdatedAt)
	if accID.Valid { l.AccountID = &accID.Int64 }
	if catID.Valid { l.CategoryID = &catID.Int64 }
	l.IsActive = active == 1
	return l, err
}

func (r *LoanRepo) List(ctx context.Context, activeOnly bool) ([]models.Loan, error) {
	q := "SELECT " + loanCols + " FROM loans"
	if activeOnly { q += " WHERE is_active=1" }
	q += " ORDER BY start_date DESC"

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil { return nil, err }
	defer rows.Close()

	out := make([]models.Loan, 0)
	for rows.Next() {
		l, err := scanLoan(rows)
		if err != nil { return nil, err }
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *LoanRepo) GetByID(ctx context.Context, id int64) (*models.Loan, error) {
	l, err := scanLoan(r.db.QueryRowContext(ctx, "SELECT "+loanCols+" FROM loans WHERE id=?", id))
	if err == sql.ErrNoRows { return nil, nil }
	if err != nil { return nil, err }
	return &l, nil
}

func (r *LoanRepo) Create(ctx context.Context, in models.CreateLoanInput) (*models.Loan, error) {
	now := ts()
	pmt := models.CalcMonthlyPayment(in.Principal, in.AnnualRate, in.TermMonths)
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO loans (name,principal,annual_rate,term_months,start_date,monthly_payment,already_paid,account_id,category_id,is_active,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,1,?,?)`,
		in.Name, in.Principal, in.AnnualRate, in.TermMonths, in.StartDate, pmt,
		in.AlreadyPaid, in.AccountID, in.CategoryID, now, now)
	if err != nil { return nil, err }
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *LoanRepo) Update(ctx context.Context, id int64, in models.UpdateLoanInput) (*models.Loan, error) {
	now := ts()
	_, err := r.db.ExecContext(ctx,
		`UPDATE loans SET name=?, annual_rate=?, account_id=?, category_id=?, is_active=?, updated_at=? WHERE id=?`,
		in.Name, in.AnnualRate, in.AccountID, in.CategoryID, boolInt(in.IsActive), now, id)
	if err != nil { return nil, err }
	return r.GetByID(ctx, id)
}

func (r *LoanRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM loans WHERE id=?", id)
	return err
}

func (r *LoanRepo) GetDailySchedule(ctx context.Context, id int64, from, to string) (*models.LoanDailySchedule, error) {
	loan, err := r.GetByID(ctx, id)
	if err != nil || loan == nil { return nil, err }

	// get all payments linked to this loan
	rows, err := r.db.QueryContext(ctx, txBase+" WHERE t.loan_id = ? ORDER BY t.date", id)
	if err != nil { return nil, err }
	defer rows.Close()

	var payments []models.Transaction
	for rows.Next() {
		t, err := scanTx(rows)
		if err != nil { return nil, err }
		payments = append(payments, t)
	}
	if err := rows.Err(); err != nil { return nil, err }

	return models.BuildDailySchedule(*loan, payments, from, to), nil
}