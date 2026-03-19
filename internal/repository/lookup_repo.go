package repository

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"unicode"

	"homebudget/internal/models"
)

type LookupRepo struct{ db *sql.DB }

func NewLookupRepo(db *sql.DB) *LookupRepo { return &LookupRepo{db: db} }

func (r *LookupRepo) ListByGroup(ctx context.Context, group string, activeOnly bool) ([]models.LookupValue, error) {
	q := "SELECT id, group_name, value, label, sort_order, is_active FROM lookup_values WHERE group_name = ?"
	if activeOnly {
		q += " AND is_active = 1"
	}
	q += " ORDER BY sort_order, value"

	rows, err := r.db.QueryContext(ctx, q, group)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.LookupValue, 0)
	for rows.Next() {
		var v models.LookupValue
		var active int
		if err := rows.Scan(&v.ID, &v.GroupName, &v.Value, &v.Label, &v.SortOrder, &active); err != nil {
			return nil, err
		}
		v.IsActive = active == 1
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *LookupRepo) ListAll(ctx context.Context) ([]models.LookupValue, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, group_name, value, label, sort_order, is_active FROM lookup_values ORDER BY group_name, sort_order, value")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.LookupValue, 0)
	for rows.Next() {
		var v models.LookupValue
		var active int
		if err := rows.Scan(&v.ID, &v.GroupName, &v.Value, &v.Label, &v.SortOrder, &active); err != nil {
			return nil, err
		}
		v.IsActive = active == 1
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *LookupRepo) GetByID(ctx context.Context, id int64) (*models.LookupValue, error) {
	var v models.LookupValue
	var active int
	err := r.db.QueryRowContext(ctx,
		"SELECT id, group_name, value, label, sort_order, is_active FROM lookup_values WHERE id = ?", id,
	).Scan(&v.ID, &v.GroupName, &v.Value, &v.Label, &v.SortOrder, &active)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v.IsActive = active == 1
	return &v, nil
}

func sanitizeLookupValue(label string) string {
	s := strings.TrimSpace(strings.ToLower(label))

	isLatinish := true
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '_' {
			continue
		}
		isLatinish = false
		break
	}

	var tr = map[rune]string{
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d",
		'е': "e", 'ё': "e", 'ж': "zh", 'з': "z", 'и': "i",
		'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n",
		'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t",
		'у': "u", 'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch",
		'ш': "sh", 'щ': "sch", 'ъ': "", 'ы': "y", 'ь': "",
		'э': "e", 'ю': "yu", 'я': "ya",
	}

	var b strings.Builder

	if isLatinish {
		for _, r := range s {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			} else {
				b.WriteRune('_')
			}
		}
	} else {
		for _, r := range s {
			switch {
			case unicode.IsDigit(r):
				b.WriteRune(r)
			case r >= 'a' && r <= 'z':
				b.WriteRune(r)
			case tr[r] != "":
				b.WriteString(tr[r])
			case unicode.IsSpace(r) || r == '-' || r == '_':
				b.WriteRune('_')
			}
		}
	}

	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	out = strings.Trim(out, "_")
	if out == "" {
		out = "value"
	}
	return out
}

func (r *LookupRepo) nextUniqueValue(ctx context.Context, groupName, base string) (string, error) {
	value := base
	i := 2
	for {
		var cnt int
		err := r.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM lookup_values WHERE group_name = ? AND value = ?",
			groupName, value,
		).Scan(&cnt)
		if err != nil {
			return "", err
		}
		if cnt == 0 {
			return value, nil
		}
		value = base + "_" + strconv.Itoa(i)
		i++
	}
}

func (r *LookupRepo) nextSortOrder(ctx context.Context, groupName string) (int, error) {
	var max sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		"SELECT MAX(sort_order) FROM lookup_values WHERE group_name = ?",
		groupName,
	).Scan(&max)
	if err != nil {
		return 0, err
	}
	if !max.Valid {
		return 10, nil
	}
	return int(max.Int64) + 10, nil
}

func (r *LookupRepo) Create(ctx context.Context, in models.CreateLookupInput) (*models.LookupValue, error) {
	base := sanitizeLookupValue(in.Label)
	value, err := r.nextUniqueValue(ctx, in.GroupName, base)
	if err != nil {
		return nil, err
	}

	sortOrder, err := r.nextSortOrder(ctx, in.GroupName)
	if err != nil {
		return nil, err
	}

	res, err := r.db.ExecContext(ctx,
		"INSERT INTO lookup_values (group_name, value, label, sort_order) VALUES (?,?,?,?)",
		in.GroupName, value, in.Label, sortOrder)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *LookupRepo) Update(ctx context.Context, id int64, in models.UpdateLookupInput) (*models.LookupValue, error) {
	active := 0
	if in.IsActive {
		active = 1
	}
	_, err := r.db.ExecContext(ctx,
		"UPDATE lookup_values SET label=?, is_active=? WHERE id=?",
		in.Label, active, id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *LookupRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM lookup_values WHERE id=?", id)
	return err
}

func (r *LookupRepo) GetMeta(ctx context.Context) (*models.Meta, error) {
	all, err := r.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	m := &models.Meta{
		AccountTypes:     make([]models.LookupValue, 0),
		Currencies:       make([]models.LookupValue, 0),
		TransactionTypes: make([]models.LookupValue, 0),
		CategoryTypes:    make([]models.LookupValue, 0),
		RecurrenceTypes:  make([]models.LookupValue, 0),
	}

	for _, v := range all {
		if !v.IsActive {
			continue
		}
		switch v.GroupName {
		case "account_type":
			m.AccountTypes = append(m.AccountTypes, v)
		case "currency":
			m.Currencies = append(m.Currencies, v)
		case "transaction_type":
			m.TransactionTypes = append(m.TransactionTypes, v)
		case "category_type":
			m.CategoryTypes = append(m.CategoryTypes, v)
		case "recurrence":
			m.RecurrenceTypes = append(m.RecurrenceTypes, v)
		}
	}
	return m, nil
}