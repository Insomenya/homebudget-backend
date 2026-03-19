package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"

	"homebudget/internal/models"
)

type AnalyticsRepo struct{ db *sql.DB }

func NewAnalyticsRepo(db *sql.DB) *AnalyticsRepo { return &AnalyticsRepo{db: db} }

func (r *AnalyticsRepo) CategoryBreakdown(ctx context.Context, f models.AnalyticsFilter) (*models.CategoryBreakdown, error) {
	txType := f.Type
	if txType == "" {
		txType = "expense"
	}

	conds := []string{"t.type = ?", "t.is_pending = 0"}
	args := []interface{}{txType}

	if f.DateFrom != "" {
		conds = append(conds, "t.date >= ?")
		args = append(args, f.DateFrom)
	}
	if f.DateTo != "" {
		conds = append(conds, "t.date <= ?")
		args = append(args, f.DateTo)
	}
	if f.AccountID != nil {
		conds = append(conds, "t.account_id = ?")
		args = append(args, *f.AccountID)
	}
	if f.CategoryID != nil {
		conds = append(conds, "(t.category_id = ? OR c.parent_id = ?)")
		args = append(args, *f.CategoryID, *f.CategoryID)
	}
	if f.SharedGroupID != nil {
		conds = append(conds, "t.shared_group_id = ?")
		args = append(args, *f.SharedGroupID)
	}

	where := strings.Join(conds, " AND ")

	q := fmt.Sprintf(`
		SELECT c.id, c.name, c.icon, c.parent_id, COALESCE(SUM(t.amount), 0)
		FROM transactions t
		JOIN categories c ON c.id = t.category_id
		WHERE %s
		GROUP BY c.id, c.name, c.icon, c.parent_id
		ORDER BY SUM(t.amount) DESC`, where)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.CategorySlice, 0)
	var total float64

	for rows.Next() {
		var cs models.CategorySlice
		var pid sql.NullInt64
		if err := rows.Scan(&cs.CategoryID, &cs.CategoryName, &cs.CategoryIcon, &pid, &cs.Amount); err != nil {
			return nil, err
		}
		if pid.Valid {
			cs.ParentID = &pid.Int64
		}
		total += cs.Amount
		items = append(items, cs)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range items {
		if total > 0 {
			items[i].Percentage = math.Round(items[i].Amount/total*10000) / 100
		}
	}

	return &models.CategoryBreakdown{
		Type: txType, Total: math.Round(total*100) / 100, Items: items,
	}, nil
}

func (r *AnalyticsRepo) Trends(ctx context.Context, f models.AnalyticsFilter) (*models.TrendData, error) {
	gran := f.Granularity
	if gran == "" {
		gran = "month"
	}
	format := granFormat(gran)

	conds := []string{"1=1", "t.is_pending = 0"}
	var args []interface{}

	if f.DateFrom != "" {
		conds = append(conds, "t.date >= ?")
		args = append(args, f.DateFrom)
	}
	if f.DateTo != "" {
		conds = append(conds, "t.date <= ?")
		args = append(args, f.DateTo)
	}
	if f.AccountID != nil {
		conds = append(conds, "(t.account_id = ? OR t.to_account_id = ?)")
		args = append(args, *f.AccountID, *f.AccountID)
	}
	if f.CategoryID != nil {
		conds = append(conds, "t.category_id = ?")
		args = append(args, *f.CategoryID)
	}
	if f.SharedGroupID != nil {
		conds = append(conds, "t.shared_group_id = ?")
		args = append(args, *f.SharedGroupID)
	}

	where := strings.Join(conds, " AND ")

	q := fmt.Sprintf(`
		SELECT %s as period,
		       COALESCE(SUM(CASE WHEN t.type='income' THEN t.amount ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN t.type='expense' THEN t.amount ELSE 0 END), 0)
		FROM transactions t
		WHERE %s AND t.type IN ('income','expense')
		GROUP BY period
		ORDER BY period`, format, where)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]models.TrendPoint, 0)
	for rows.Next() {
		var p models.TrendPoint
		if err := rows.Scan(&p.Period, &p.Income, &p.Expenses); err != nil {
			return nil, err
		}
		p.Income = math.Round(p.Income*100) / 100
		p.Expenses = math.Round(p.Expenses*100) / 100
		p.Net = math.Round((p.Income-p.Expenses)*100) / 100
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &models.TrendData{
		Granularity: gran,
		DateFrom:    f.DateFrom,
		DateTo:      f.DateTo,
		Points:      points,
	}, nil
}

func granFormat(g string) string {
	switch g {
	case "day":
		return "strftime('%Y-%m-%d', t.date)"
	case "week":
		return "strftime('%Y-W%W', t.date)"
	case "year":
		return "strftime('%Y', t.date)"
	default:
		return "strftime('%Y-%m', t.date)"
	}
}