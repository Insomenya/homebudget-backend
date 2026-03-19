package repository

import (
	"context"
	"database/sql"

	"homebudget/internal/models"
)

type AccountRepo struct{ db *sql.DB }

func NewAccountRepo(db *sql.DB) *AccountRepo { return &AccountRepo{db: db} }

const accountCols = `id, name, type, currency, initial_balance, member_id, is_archived, created_at, updated_at`

func scanAccount(s scannable) (models.Account, error) {
	var a models.Account
	var arch int
	err := s.Scan(&a.ID, &a.Name, &a.Type, &a.Currency,
		&a.InitialBalance, &a.MemberID, &arch, &a.CreatedAt, &a.UpdatedAt)
	a.IsArchived = arch == 1
	return a, err
}

func (r *AccountRepo) List(ctx context.Context, inclArch bool) ([]models.Account, error) {
	q := "SELECT " + accountCols + " FROM accounts"
	if !inclArch {
		q += " WHERE is_archived=0"
	}
	q += " ORDER BY id"
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AccountRepo) GetByID(ctx context.Context, id int64) (*models.Account, error) {
	a, err := scanAccount(r.db.QueryRowContext(ctx,
		"SELECT "+accountCols+" FROM accounts WHERE id=?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AccountRepo) Create(ctx context.Context, in models.CreateAccountInput) (*models.Account, error) {
	now := ts()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO accounts (name,type,currency,initial_balance,member_id,is_archived,created_at,updated_at)
		 VALUES (?,?,?,?,?,0,?,?)`,
		in.Name, in.Type, in.Currency, in.InitialBalance, in.MemberID, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *AccountRepo) Update(ctx context.Context, id int64, in models.UpdateAccountInput) (*models.Account, error) {
	now := ts()
	_, err := r.db.ExecContext(ctx,
		`UPDATE accounts SET name=?,type=?,currency=?,initial_balance=?,member_id=?,is_archived=?,updated_at=?
		 WHERE id=?`,
		in.Name, in.Type, in.Currency, in.InitialBalance, in.MemberID,
		boolInt(in.IsArchived), now, id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *AccountRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM accounts WHERE id=?", id)
	return err
}

// ListWithBalances — два отдельных запроса вместо коррелированных подзапросов.
func (r *AccountRepo) ListWithBalances(ctx context.Context) ([]models.AccountBalance, error) {
	// 1. Загрузить все счета
	accounts, err := r.List(ctx, false)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return []models.AccountBalance{}, nil
	}

	// 2. Посчитать все движения по счетам одним запросом
	type movement struct {
		income   float64
		expense  float64
		xferIn   float64
		xferOut  float64
	}
	moves := make(map[int64]*movement)

	// доходы
	rows, err := r.db.QueryContext(ctx,
		`SELECT account_id, SUM(amount) FROM transactions
		 WHERE account_id IS NOT NULL AND type='income' AND is_pending=0
		 GROUP BY account_id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var sum float64
		if err := rows.Scan(&id, &sum); err != nil {
			rows.Close()
			return nil, err
		}
		if moves[id] == nil {
			moves[id] = &movement{}
		}
		moves[id].income = sum
	}
	rows.Close()

	// расходы
	rows, err = r.db.QueryContext(ctx,
		`SELECT account_id, SUM(amount) FROM transactions
		 WHERE account_id IS NOT NULL AND type='expense' AND is_pending=0
		 GROUP BY account_id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var sum float64
		if err := rows.Scan(&id, &sum); err != nil {
			rows.Close()
			return nil, err
		}
		if moves[id] == nil {
			moves[id] = &movement{}
		}
		moves[id].expense = sum
	}
	rows.Close()

	// переводы — исходящие
	rows, err = r.db.QueryContext(ctx,
		`SELECT account_id, SUM(amount) FROM transactions
		 WHERE account_id IS NOT NULL AND type='transfer' AND is_pending=0
		 GROUP BY account_id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var sum float64
		if err := rows.Scan(&id, &sum); err != nil {
			rows.Close()
			return nil, err
		}
		if moves[id] == nil {
			moves[id] = &movement{}
		}
		moves[id].xferOut = sum
	}
	rows.Close()

	// переводы — входящие
	rows, err = r.db.QueryContext(ctx,
		`SELECT to_account_id, SUM(amount) FROM transactions
		 WHERE to_account_id IS NOT NULL AND type='transfer' AND is_pending=0
		 GROUP BY to_account_id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var sum float64
		if err := rows.Scan(&id, &sum); err != nil {
			rows.Close()
			return nil, err
		}
		if moves[id] == nil {
			moves[id] = &movement{}
		}
		moves[id].xferIn = sum
	}
	rows.Close()

	// 3. Собрать результат
	out := make([]models.AccountBalance, 0, len(accounts))
	for _, a := range accounts {
		m := moves[a.ID]
		balance := a.InitialBalance
		if m != nil {
			balance += m.income - m.expense + m.xferIn - m.xferOut
		}
		out = append(out, models.AccountBalance{
			Account:        a,
			CurrentBalance: balance,
		})
	}

	return out, nil
}