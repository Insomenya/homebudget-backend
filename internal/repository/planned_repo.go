package repository

import (
	"context"
	"database/sql"
	"time"

	"homebudget/internal/models"
)

type PlannedRepo struct{ db *sql.DB }

func NewPlannedRepo(db *sql.DB) *PlannedRepo { return &PlannedRepo{db: db} }

const ptCols = `id, name, amount, type, account_id, category_id,
	shared_group_id, paid_by_member_id, recurrence,
	start_date, end_date, next_due, notify_days,
	is_auto, is_active, created_at, updated_at`

func scanPT(s scannable) (models.PlannedTransaction, error) {
	var p models.PlannedTransaction
	var accID, catID, grpID, paidID sql.NullInt64
	var endDate sql.NullString
	var isAuto, isActive int

	err := s.Scan(
		&p.ID, &p.Name, &p.Amount, &p.Type,
		&accID, &catID, &grpID, &paidID,
		&p.Recurrence, &p.StartDate, &endDate, &p.NextDue,
		&p.NotifyDays, &isAuto, &isActive,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if accID.Valid {
		p.AccountID = &accID.Int64
	}
	if catID.Valid {
		p.CategoryID = &catID.Int64
	}
	if grpID.Valid {
		p.SharedGroupID = &grpID.Int64
	}
	if paidID.Valid {
		p.PaidByMemberID = &paidID.Int64
	}
	if endDate.Valid {
		p.EndDate = &endDate.String
	}
	p.IsAuto = isAuto == 1
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
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO planned_transactions
		 (name,amount,type,account_id,category_id,shared_group_id,paid_by_member_id,
		  recurrence,start_date,end_date,next_due,notify_days,is_auto,is_active,
		  created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,1,?,?)`,
		in.Name, in.Amount, in.Type, in.AccountID, in.CategoryID,
		in.SharedGroupID, in.PaidByMemberID,
		in.Recurrence, in.StartDate, in.EndDate, in.StartDate,
		in.NotifyDays, boolInt(in.IsAuto), now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *PlannedRepo) Update(ctx context.Context, id int64, in models.UpdatePlannedInput) (*models.PlannedTransaction, error) {
	now := ts()
	_, err := r.db.ExecContext(ctx,
		`UPDATE planned_transactions
		 SET name=?,amount=?,type=?,account_id=?,category_id=?,
		     shared_group_id=?,paid_by_member_id=?,
		     recurrence=?,start_date=?,end_date=?,next_due=?,
		     notify_days=?,is_auto=?,updated_at=?
		 WHERE id=?`,
		in.Name, in.Amount, in.Type, in.AccountID, in.CategoryID,
		in.SharedGroupID, in.PaidByMemberID,
		in.Recurrence, in.StartDate, in.EndDate, in.StartDate,
		in.NotifyDays, boolInt(in.IsAuto), now, id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *PlannedRepo) Delete(ctx context.Context, id int64) error {
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

func (r *PlannedRepo) AdvanceNextDue(ctx context.Context, id int64) error {
	pt, err := r.GetByID(ctx, id)
	if err != nil || pt == nil {
		return err
	}

	next, active := models.AdvanceDate(pt.NextDue, pt.Recurrence, pt.EndDate)
	now := ts()
	_, err = r.db.ExecContext(ctx,
		"UPDATE planned_transactions SET next_due=?, is_active=?, updated_at=? WHERE id=?",
		next, boolInt(active), now, id)
	return err
}

// MaterializeDue создаёт pending-транзакции для отложенных платежей
// у которых next_due <= сегодня + notify_days. Возвращает количество созданных.
func (r *PlannedRepo) MaterializeDue(ctx context.Context, txRepo *TransactionRepo) int {
	active, err := r.List(ctx, true)
	if err != nil {
		return 0
	}

	created := 0
	for _, pt := range active {
		if !pt.IsActive {
			continue
		}
		cutoff := time.Now().AddDate(0, 0, pt.NotifyDays).Format("2006-01-02")
		if pt.NextDue > cutoff {
			continue
		}

		exists, err := txRepo.ExistsPendingForPlanned(ctx, pt.ID, pt.NextDue)
		if err != nil || exists {
			continue
		}

		txIn := models.CreateTransactionInput{
			Date:           pt.NextDue,
			Amount:         pt.Amount,
			Description:    pt.Name,
			Type:           pt.Type,
			AccountID:      pt.AccountID,
			CategoryID:     pt.CategoryID,
			SharedGroupID:  pt.SharedGroupID,
			PaidByMemberID: pt.PaidByMemberID,
			IsPending:      true,
			PlannedID:      &pt.ID,
		}

		if msg := txIn.Validate(); msg != "" {
			continue
		}

		if _, err := txRepo.Create(ctx, txIn); err != nil {
			continue
		}
		created++

		_ = r.AdvanceNextDue(ctx, pt.ID)
	}
	return created
}