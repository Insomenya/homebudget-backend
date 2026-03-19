package repository

import (
	"context"
	"database/sql"

	"homebudget/internal/models"
)

type CategoryRepo struct{ db *sql.DB }

func NewCategoryRepo(db *sql.DB) *CategoryRepo { return &CategoryRepo{db: db} }

const catCols = `id, name, type, icon, parent_id, sort_order, is_archived, created_at, updated_at`

func scanCategory(s scannable) (models.Category, error) {
	var c models.Category
	var pid sql.NullInt64
	var arch int
	err := s.Scan(&c.ID, &c.Name, &c.Type, &c.Icon,
		&pid, &c.SortOrder, &arch, &c.CreatedAt, &c.UpdatedAt)
	if pid.Valid {
		c.ParentID = &pid.Int64
	}
	c.IsArchived = arch == 1
	return c, err
}

func (r *CategoryRepo) List(ctx context.Context, inclArch bool) ([]models.Category, error) {
	q := "SELECT " + catCols + " FROM categories"
	if !inclArch {
		q += " WHERE is_archived=0"
	}
	q += " ORDER BY type, sort_order, name"
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CategoryRepo) GetByID(ctx context.Context, id int64) (*models.Category, error) {
	c, err := scanCategory(r.db.QueryRowContext(ctx,
		"SELECT "+catCols+" FROM categories WHERE id=?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepo) Create(ctx context.Context, in models.CreateCategoryInput) (*models.Category, error) {
	now := ts()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO categories (name,type,icon,parent_id,sort_order,is_archived,created_at,updated_at)
		 VALUES (?,?,?,?,?,0,?,?)`,
		in.Name, in.Type, in.Icon, in.ParentID, in.SortOrder, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *CategoryRepo) Update(ctx context.Context, id int64, in models.UpdateCategoryInput) (*models.Category, error) {
	now := ts()
	_, err := r.db.ExecContext(ctx,
		`UPDATE categories SET name=?,type=?,icon=?,parent_id=?,sort_order=?,is_archived=?,updated_at=?
		 WHERE id=?`,
		in.Name, in.Type, in.Icon, in.ParentID, in.SortOrder,
		boolInt(in.IsArchived), now, id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *CategoryRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM categories WHERE id=?", id)
	return err
}