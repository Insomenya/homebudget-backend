// FILE: internal/repository/transaction_repo.go
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"homebudget/internal/models"
)

type TransactionRepo struct {
	db      *sql.DB
	loanCb  func(ctx context.Context, loanID int64) // callback to recalc loan after tx change
}

func NewTransactionRepo(db *sql.DB) *TransactionRepo { return &TransactionRepo{db: db} }

// SetLoanCallback sets a callback that is called after a transaction
// with loan_id is created/updated/deleted, to recalculate the loan.
func (r *TransactionRepo) SetLoanCallback(cb func(ctx context.Context, loanID int64)) {
	r.loanCb = cb
}

const txBase = `SELECT t.id, t.date, t.amount, t.description, t.type,
	t.account_id, t.to_account_id, t.category_id,
	t.shared_group_id, t.paid_by_member_id, t.loan_id,
	t.reminder_id,
	t.created_at, t.updated_at
	FROM transactions t`

func scanTx(s scannable) (models.Transaction, error) {
	var t models.Transaction
	var accID, toAccID, catID, grpID, paidID, loanID, reminderID sql.NullInt64
	err := s.Scan(
		&t.ID, &t.Date, &t.Amount, &t.Description, &t.Type,
		&accID, &toAccID, &catID, &grpID, &paidID, &loanID,
		&reminderID,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if accID.Valid {
		t.AccountID = &accID.Int64
	}
	if toAccID.Valid {
		t.ToAccountID = &toAccID.Int64
	}
	if catID.Valid {
		t.CategoryID = &catID.Int64
	}
	if grpID.Valid {
		t.SharedGroupID = &grpID.Int64
	}
	if paidID.Valid {
		t.PaidByMemberID = &paidID.Int64
	}
	if loanID.Valid {
		t.LoanID = &loanID.Int64
	}
	if reminderID.Valid {
		t.ReminderID = &reminderID.Int64
	}
	return t, err
}

func (r *TransactionRepo) GetByID(ctx context.Context, id int64) (*models.Transaction, error) {
	t, err := scanTx(r.db.QueryRowContext(ctx, txBase+" WHERE t.id=?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TransactionRepo) Create(ctx context.Context, in models.CreateTransactionInput) (*models.Transaction, error) {
	now := ts()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO transactions
		 (date,amount,description,type,account_id,to_account_id,
		  category_id,shared_group_id,paid_by_member_id,loan_id,
		  reminder_id,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		in.Date, in.Amount, in.Description, in.Type,
		in.AccountID, in.ToAccountID, in.CategoryID,
		in.SharedGroupID, in.PaidByMemberID, in.LoanID,
		in.ReminderID, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	// Recalc loan if this transaction is linked to one
	if in.LoanID != nil && r.loanCb != nil {
		r.loanCb(ctx, *in.LoanID)
	}

	return r.GetByID(ctx, id)
}

func (r *TransactionRepo) Update(ctx context.Context, id int64, in models.UpdateTransactionInput) (*models.Transaction, error) {
	// Get old transaction to check if loan changed
	old, _ := r.GetByID(ctx, id)

	now := ts()
	_, err := r.db.ExecContext(ctx,
		`UPDATE transactions
		 SET date=?,amount=?,description=?,type=?,
		     account_id=?,to_account_id=?,category_id=?,
		     shared_group_id=?,paid_by_member_id=?,loan_id=?,
		     reminder_id=?,updated_at=?
		 WHERE id=?`,
		in.Date, in.Amount, in.Description, in.Type,
		in.AccountID, in.ToAccountID, in.CategoryID,
		in.SharedGroupID, in.PaidByMemberID, in.LoanID,
		in.ReminderID, now, id)
	if err != nil {
		return nil, err
	}

	// Recalc affected loans
	if r.loanCb != nil {
		if old != nil && old.LoanID != nil {
			r.loanCb(ctx, *old.LoanID)
		}
		if in.LoanID != nil && (old == nil || old.LoanID == nil || *old.LoanID != *in.LoanID) {
			r.loanCb(ctx, *in.LoanID)
		}
	}

	return r.GetByID(ctx, id)
}

func (r *TransactionRepo) Delete(ctx context.Context, id int64) error {
	// Get transaction before deleting to know its loan_id
	old, _ := r.GetByID(ctx, id)

	_, err := r.db.ExecContext(ctx, "DELETE FROM transactions WHERE id=?", id)
	if err != nil {
		return err
	}

	// Recalc loan
	if old != nil && old.LoanID != nil && r.loanCb != nil {
		r.loanCb(ctx, *old.LoanID)
	}

	return nil
}

func (r *TransactionRepo) List(ctx context.Context, f models.TransactionFilter) (*models.TransactionList, error) {
	where, args := buildTxWhere(f)

	var total int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM transactions t WHERE "+where, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	page, limit := normPage(f.Page, f.Limit)
	pages := (total + limit - 1) / limit
	if pages == 0 {
		pages = 1
	}
	offset := (page - 1) * limit
	order := txOrder(f.SortBy, f.SortDir)

	q := fmt.Sprintf("%s WHERE %s ORDER BY %s LIMIT ? OFFSET ?",
		txBase, where, order)

	rows, err := r.db.QueryContext(ctx, q, append(args, limit, offset)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.Transaction, 0)
	for rows.Next() {
		t, err := scanTx(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &models.TransactionList{
		Items: items, Total: total,
		Page: page, Limit: limit, Pages: pages,
	}, nil
}

func (r *TransactionRepo) PeriodSummary(ctx context.Context, from, to string) (*models.PeriodSummary, error) {
	var income, expenses float64

	err := r.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(amount),0) FROM transactions WHERE type='income' AND date>=? AND date<=?",
		from, to).Scan(&income)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(amount),0) FROM transactions WHERE type='expense' AND date>=? AND date<=?",
		from, to).Scan(&expenses)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.icon, SUM(t.amount)
		FROM transactions t
		JOIN categories c ON c.id = t.category_id
		WHERE t.type='expense' AND t.date>=? AND t.date<=?
		GROUP BY c.id, c.name, c.icon
		ORDER BY SUM(t.amount) DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cats := make([]models.CategoryTotal, 0)
	for rows.Next() {
		var ct models.CategoryTotal
		if err := rows.Scan(&ct.CategoryID, &ct.CategoryName, &ct.CategoryIcon, &ct.Amount); err != nil {
			return nil, err
		}
		cats = append(cats, ct)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &models.PeriodSummary{
		TotalIncome:   income,
		TotalExpenses: expenses,
		ByCategory:    cats,
	}, nil
}

func (r *TransactionRepo) Recent(ctx context.Context, limit int) ([]models.Transaction, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.QueryContext(ctx,
		txBase+" ORDER BY t.date DESC, t.id DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Transaction, 0)
	for rows.Next() {
		t, err := scanTx(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func buildTxWhere(f models.TransactionFilter) (string, []interface{}) {
	c := []string{"1=1"}
	var a []interface{}
	if f.DateFrom != "" {
		c = append(c, "t.date >= ?")
		a = append(a, f.DateFrom)
	}
	if f.DateTo != "" {
		c = append(c, "t.date <= ?")
		a = append(a, f.DateTo)
	}
	if f.Search != "" {
		c = append(c, "t.description LIKE ?")
		a = append(a, "%"+f.Search+"%")
	}
	if f.Type != "" {
		c = append(c, "t.type = ?")
		a = append(a, f.Type)
	}
	if f.AccountID != nil {
		c = append(c, "(t.account_id=? OR t.to_account_id=?)")
		a = append(a, *f.AccountID, *f.AccountID)
	}
	if f.CategoryID != nil {
		c = append(c, "t.category_id=?")
		a = append(a, *f.CategoryID)
	}
	if f.SharedGroupID != nil {
		c = append(c, "t.shared_group_id=?")
		a = append(a, *f.SharedGroupID)
	}
	if f.PaidByMemberID != nil {
		c = append(c, "t.paid_by_member_id=?")
		a = append(a, *f.PaidByMemberID)
	}
	if f.LoanID != nil {
		c = append(c, "t.loan_id=?")
		a = append(a, *f.LoanID)
	}
	if f.IsShared != nil {
		if *f.IsShared {
			c = append(c, "t.shared_group_id IS NOT NULL")
		} else {
			c = append(c, "t.shared_group_id IS NULL")
		}
	}
	return strings.Join(c, " AND "), a
}

var txSortCols = map[string]string{
	"date": "t.date", "amount": "t.amount",
	"type": "t.type", "created_at": "t.created_at",
	"description": "t.description",
}

func txOrder(col, dir string) string {
	m, ok := txSortCols[col]
	if !ok {
		m = "t.date"
	}
	if strings.ToUpper(dir) == "ASC" {
		return m + " ASC, t.id ASC"
	}
	return m + " DESC, t.id DESC"
}

func normPage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}