package repository

import (
	"context"
	"database/sql"

	"homebudget/internal/models"
)

type BudgetRepo struct{ db *sql.DB }

func NewBudgetRepo(db *sql.DB) *BudgetRepo { return &BudgetRepo{db: db} }

// ── Columns ─────────────────────────────────────────

func (r *BudgetRepo) ListColumns(ctx context.Context) ([]models.BudgetColumn, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, name, col_type, ref_id, sort_order FROM budget_columns ORDER BY sort_order")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.BudgetColumn, 0)
	for rows.Next() {
		var c models.BudgetColumn
		var refID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Name, &c.ColType, &refID, &c.SortOrder); err != nil {
			return nil, err
		}
		if refID.Valid {
			c.RefID = &refID.Int64
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *BudgetRepo) CreateColumn(ctx context.Context, in models.CreateBudgetColumnInput) (*models.BudgetColumn, error) {
	var maxOrder int
	r.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(sort_order),0) FROM budget_columns").Scan(&maxOrder)

	res, err := r.db.ExecContext(ctx,
		"INSERT INTO budget_columns (name, col_type, ref_id, sort_order) VALUES (?,?,?,?)",
		in.Name, in.ColType, in.RefID, maxOrder+10)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	var c models.BudgetColumn
	var refID sql.NullInt64
	r.db.QueryRowContext(ctx,
		"SELECT id, name, col_type, ref_id, sort_order FROM budget_columns WHERE id=?", id).
		Scan(&c.ID, &c.Name, &c.ColType, &refID, &c.SortOrder)
	if refID.Valid {
		c.RefID = &refID.Int64
	}
	return &c, nil
}

func (r *BudgetRepo) DeleteColumn(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM budget_columns WHERE id=?", id)
	return err
}

// ── Rows + Cells ────────────────────────────────────

func (r *BudgetRepo) GetTable(ctx context.Context, page, limit int) (*models.BudgetTable, error) {
	columns, err := r.ListColumns(ctx)
	if err != nil {
		return nil, err
	}

	var total int
	r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM budget_rows").Scan(&total)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 30
	}
	pages := (total + limit - 1) / limit
	if pages == 0 {
		pages = 1
	}
	offset := (page - 1) * limit

	rowsQ, err := r.db.QueryContext(ctx,
		"SELECT id, date, label, is_executed FROM budget_rows ORDER BY date, id LIMIT ? OFFSET ?",
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rowsQ.Close()

	budgetRows := make([]models.BudgetRow, 0)
	var rowIDs []int64
	for rowsQ.Next() {
		var br models.BudgetRow
		var exec int
		if err := rowsQ.Scan(&br.ID, &br.Date, &br.Label, &exec); err != nil {
			return nil, err
		}
		br.IsExecuted = exec == 1
		br.Cells = make(map[int64]float64)
		budgetRows = append(budgetRows, br)
		rowIDs = append(rowIDs, br.ID)
	}
	if err := rowsQ.Err(); err != nil {
		return nil, err
	}

	// load cells
	if len(rowIDs) > 0 {
		cellRows, err := r.db.QueryContext(ctx,
			"SELECT row_id, column_id, amount FROM budget_cells WHERE row_id IN ("+placeholders(len(rowIDs))+")",
			toAny(rowIDs)...)
		if err != nil {
			return nil, err
		}
		defer cellRows.Close()

		rowMap := make(map[int64]int)
		for i, br := range budgetRows {
			rowMap[br.ID] = i
		}

		for cellRows.Next() {
			var rowID, colID int64
			var amount float64
			if err := cellRows.Scan(&rowID, &colID, &amount); err != nil {
				return nil, err
			}
			if idx, ok := rowMap[rowID]; ok {
				budgetRows[idx].Cells[colID] = amount
			}
		}
		if err := cellRows.Err(); err != nil {
			return nil, err
		}
	}

	return &models.BudgetTable{
		Columns: columns,
		Rows:    budgetRows,
		Total:   total,
		Page:    page,
		Pages:   pages,
	}, nil
}

func (r *BudgetRepo) CreateRow(ctx context.Context, in models.CreateBudgetRowInput) (*models.BudgetRow, error) {
	res, err := r.db.ExecContext(ctx,
		"INSERT INTO budget_rows (date, label, is_executed) VALUES (?,?,0)",
		in.Date, in.Label)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	return &models.BudgetRow{
		ID:    id,
		Date:  in.Date,
		Label: in.Label,
		Cells: make(map[int64]float64),
	}, nil
}

func (r *BudgetRepo) DeleteRow(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM budget_rows WHERE id=?", id)
	return err
}

func (r *BudgetRepo) UpdateCell(ctx context.Context, in models.UpdateBudgetCellInput) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO budget_cells (row_id, column_id, amount) VALUES (?,?,?)
		 ON CONFLICT(row_id, column_id) DO UPDATE SET amount=excluded.amount`,
		in.RowID, in.ColumnID, in.Amount)
	return err
}

func (r *BudgetRepo) ToggleExecuted(ctx context.Context, rowID int64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE budget_rows SET is_executed = 1 - is_executed WHERE id=?", rowID)
	return err
}

// helpers

func placeholders(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			s += ","
		}
		s += "?"
	}
	return s
}

func toAny(ids []int64) []interface{} {
	out := make([]interface{}, len(ids))
	for i, v := range ids {
		out[i] = v
	}
	return out
}