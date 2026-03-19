package repository

import (
	"context"
	"database/sql"

	"homebudget/internal/models"
)

type MemberRepo struct{ db *sql.DB }

func NewMemberRepo(db *sql.DB) *MemberRepo { return &MemberRepo{db: db} }

const memberCols = `id, name, icon, is_archived, created_at, updated_at`

func scanMember(s scannable) (models.Member, error) {
	var m models.Member
	var arch int
	err := s.Scan(&m.ID, &m.Name, &m.Icon, &arch, &m.CreatedAt, &m.UpdatedAt)
	m.IsArchived = arch == 1
	return m, err
}

func (r *MemberRepo) List(ctx context.Context, inclArch bool) ([]models.Member, error) {
	q := "SELECT " + memberCols + " FROM members"
	if !inclArch {
		q += " WHERE is_archived=0"
	}
	q += " ORDER BY id"

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Member
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MemberRepo) GetByID(ctx context.Context, id int64) (*models.Member, error) {
	m, err := scanMember(r.db.QueryRowContext(ctx,
		"SELECT "+memberCols+" FROM members WHERE id=?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MemberRepo) Create(ctx context.Context, in models.CreateMemberInput) (*models.Member, error) {
	now := ts()
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO members (name,icon,is_archived,created_at,updated_at)
		 VALUES (?,?,0,?,?)`, in.Name, in.Icon, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *MemberRepo) Update(ctx context.Context, id int64, in models.UpdateMemberInput) (*models.Member, error) {
	now := ts()
	_, err := r.db.ExecContext(ctx,
		`UPDATE members SET name=?, icon=?, is_archived=?, updated_at=? WHERE id=?`,
		in.Name, in.Icon, boolInt(in.IsArchived), now, id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *MemberRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM members WHERE id=?", id)
	return err
}