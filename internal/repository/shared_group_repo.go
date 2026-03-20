package repository

import (
	"context"
	"database/sql"
	"math"

	"homebudget/internal/models"
)

type SharedGroupRepo struct{ db *sql.DB }

func NewSharedGroupRepo(db *sql.DB) *SharedGroupRepo { return &SharedGroupRepo{db: db} }

func scanGroup(s scannable) (models.SharedGroup, error) {
	var g models.SharedGroup
	var arch int
	err := s.Scan(&g.ID, &g.Name, &g.Icon, &arch, &g.CreatedAt, &g.UpdatedAt)
	g.IsArchived = arch == 1
	return g, err
}

func (r *SharedGroupRepo) loadMembers(ctx context.Context, gid int64) ([]models.SharedGroupMember, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT sgm.id, sgm.group_id, sgm.member_id,
		        m.name, m.icon,
		        sgm.share_numerator, sgm.share_denominator
		 FROM shared_group_members sgm
		 JOIN members m ON m.id = sgm.member_id
		 WHERE sgm.group_id = ?
		 ORDER BY sgm.id`, gid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.SharedGroupMember
	for rows.Next() {
		var m models.SharedGroupMember
		if err := rows.Scan(&m.ID, &m.GroupID, &m.MemberID,
			&m.MemberName, &m.MemberIcon,
			&m.ShareNumerator, &m.ShareDenominator); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *SharedGroupRepo) withMembers(ctx context.Context, g models.SharedGroup) (*models.SharedGroupWithMembers, error) {
	members, err := r.loadMembers(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	if members == nil {
		members = []models.SharedGroupMember{}
	}
	return &models.SharedGroupWithMembers{SharedGroup: g, Members: members}, nil
}

func (r *SharedGroupRepo) List(ctx context.Context, inclArch bool) ([]models.SharedGroupWithMembers, error) {
	q := "SELECT id, name, icon, is_archived, created_at, updated_at FROM shared_groups"
	if !inclArch {
		q += " WHERE is_archived=0"
	}
	q += " ORDER BY id"
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.SharedGroupWithMembers
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		wm, err := r.withMembers(ctx, g)
		if err != nil {
			return nil, err
		}
		out = append(out, *wm)
	}
	return out, rows.Err()
}

func (r *SharedGroupRepo) GetByID(ctx context.Context, id int64) (*models.SharedGroupWithMembers, error) {
	g, err := scanGroup(r.db.QueryRowContext(ctx,
		"SELECT id, name, icon, is_archived, created_at, updated_at FROM shared_groups WHERE id=?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.withMembers(ctx, g)
}

func (r *SharedGroupRepo) saveMembers(ctx context.Context, tx *sql.Tx, gid int64, members []models.SharedGroupMemberInput) error {
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM shared_group_members WHERE group_id=?", gid); err != nil {
		return err
	}
	for _, m := range members {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO shared_group_members (group_id,member_id,share_numerator,share_denominator)
			 VALUES (?,?,?,?)`,
			gid, m.MemberID, m.ShareNumerator, m.ShareDenominator); err != nil {
			return err
		}
	}
	return nil
}

func (r *SharedGroupRepo) Create(ctx context.Context, in models.CreateSharedGroupInput) (*models.SharedGroupWithMembers, error) {
	now := ts()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO shared_groups (name,icon,is_archived,created_at,updated_at)
		 VALUES (?,?,0,?,?)`, in.Name, in.Icon, now, now)
	if err != nil {
		return nil, err
	}
	gid, _ := res.LastInsertId()
	if err := r.saveMembers(ctx, tx, gid, in.Members); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, gid)
}

func (r *SharedGroupRepo) Update(ctx context.Context, id int64, in models.UpdateSharedGroupInput) (*models.SharedGroupWithMembers, error) {
	now := ts()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`UPDATE shared_groups SET name=?, icon=?, is_archived=?, updated_at=? WHERE id=?`,
		in.Name, in.Icon, boolInt(in.IsArchived), now, id); err != nil {
		return nil, err
	}
	if err := r.saveMembers(ctx, tx, id, in.Members); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *SharedGroupRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM shared_groups WHERE id=?", id)
	return err
}

func (r *SharedGroupRepo) paidByGroup(
	ctx context.Context, groupID int64, dateFrom, dateTo string,
) (map[int64]float64, float64, error) {
	q := `SELECT paid_by_member_id, SUM(amount)
	      FROM transactions
	      WHERE shared_group_id = ? AND paid_by_member_id IS NOT NULL`
	args := []interface{}{groupID}

	if dateFrom != "" {
		q += " AND date >= ?"
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		q += " AND date <= ?"
		args = append(args, dateTo)
	}
	q += " GROUP BY paid_by_member_id"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	paidMap := make(map[int64]float64)
	var total float64
	for rows.Next() {
		var mid int64
		var amt float64
		if err := rows.Scan(&mid, &amt); err != nil {
			return nil, 0, err
		}
		paidMap[mid] = amt
		total += amt
	}
	return paidMap, total, rows.Err()
}

func (r *SharedGroupRepo) buildBalances(
	group *models.SharedGroupWithMembers, paidMap map[int64]float64, total float64,
) []models.MemberBalance {
	balances := make([]models.MemberBalance, 0, len(group.Members))
	for _, m := range group.Members {
		fair := total * m.ShareFloat()
		paid := paidMap[m.MemberID]
		pct := 0.0
		if total > 0 {
			pct = paid / total * 100
		}
		balances = append(balances, models.MemberBalance{
			MemberID:   m.MemberID,
			MemberName: m.MemberName,
			MemberIcon: m.MemberIcon,
			TotalPaid:  round2(paid),
			FairShare:  round2(fair),
			Balance:    round2(paid - fair),
			Percentage: round2(pct),
		})
	}
	return balances
}

func (r *SharedGroupRepo) GetSettlement(
	ctx context.Context, groupID int64, dateFrom, dateTo string,
) (*models.Settlement, error) {
	group, err := r.GetByID(ctx, groupID)
	if err != nil || group == nil {
		return nil, err
	}
	paidMap, total, err := r.paidByGroup(ctx, groupID, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}
	balances := r.buildBalances(group, paidMap, total)
	debts := models.ComputeDebts(balances)
	if debts == nil {
		debts = []models.Debt{}
	}
	return &models.Settlement{
		Group:         *group,
		TotalExpenses: round2(total),
		Balances:      balances,
		Debts:         debts,
	}, nil
}

func (r *SharedGroupRepo) GetTurnover(
	ctx context.Context, groupID int64, dateFrom, dateTo string,
) (*models.Turnover, error) {
	group, err := r.GetByID(ctx, groupID)
	if err != nil || group == nil {
		return nil, err
	}

	var openBal []models.MemberBalance
	if dateFrom != "" {
		pm, tot, err := r.paidByGroup(ctx, groupID, "", dateFrom)
		if err != nil {
			return nil, err
		}
		openBal = r.buildBalances(group, pm, tot)
	} else {
		openBal = zeroBalances(group)
	}

	pm, tot, err := r.paidByGroup(ctx, groupID, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}
	periodBal := r.buildBalances(group, pm, tot)
	closeBal := sumBalances(group, openBal, periodBal)

	txs, err := r.periodTransactions(ctx, groupID, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	return &models.Turnover{
		Group:           *group,
		DateFrom:        dateFrom,
		DateTo:          dateTo,
		OpeningBalances: openBal,
		Transactions:    txs,
		PeriodTotals:    periodBal,
		ClosingBalances: closeBal,
	}, nil
}

func (r *SharedGroupRepo) periodTransactions(
	ctx context.Context, gid int64, from, to string,
) ([]models.Transaction, error) {
	q := txBase + " WHERE t.shared_group_id = ?"
	args := []interface{}{gid}
	if from != "" {
		q += " AND t.date >= ?"
		args = append(args, from)
	}
	if to != "" {
		q += " AND t.date <= ?"
		args = append(args, to)
	}
	q += " ORDER BY t.date ASC, t.id ASC"

	rows, err := r.db.QueryContext(ctx, q, args...)
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

func (r *SharedGroupRepo) ListSettlementSummariesFast(
	ctx context.Context,
) ([]models.GroupSettlementSummary, error) {
	groups, err := r.List(ctx, false)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return []models.GroupSettlementSummary{}, nil
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT shared_group_id, paid_by_member_id, amount
		 FROM transactions
		 WHERE shared_group_id IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type key struct{ gid, mid int64 }
	paidMap := make(map[key]float64)
	groupTotals := make(map[int64]float64)

	for rows.Next() {
		var gid int64
		var midN sql.NullInt64
		var amt float64
		if err := rows.Scan(&gid, &midN, &amt); err != nil {
			return nil, err
		}
		if midN.Valid {
			k := key{gid, midN.Int64}
			paidMap[k] += amt
			groupTotals[gid] += amt
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]models.GroupSettlementSummary, 0, len(groups))
	for _, g := range groups {
		total := groupTotals[g.ID]
		balances := make([]models.MemberBalance, 0, len(g.Members))
		for _, m := range g.Members {
			fair := total * m.ShareFloat()
			paid := paidMap[key{g.ID, m.MemberID}]
			balances = append(balances, models.MemberBalance{
				MemberID:   m.MemberID,
				MemberName: m.MemberName,
				MemberIcon: m.MemberIcon,
				TotalPaid:  round2(paid),
				FairShare:  round2(fair),
				Balance:    round2(paid - fair),
				Percentage: round2(func() float64 {
					if total > 0 {
						return paid / total * 100
					}
					return 0
				}()),
			})
		}
		debts := models.ComputeDebts(balances)
		if debts == nil {
			debts = []models.Debt{}
		}
		out = append(out, models.GroupSettlementSummary{
			GroupID:      g.ID,
			GroupName:    g.Name,
			GroupIcon:    g.Icon,
			MemberCount: len(g.Members),
			Debts:        debts,
		})
	}
	return out, nil
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func zeroBalances(g *models.SharedGroupWithMembers) []models.MemberBalance {
	out := make([]models.MemberBalance, len(g.Members))
	for i, m := range g.Members {
		out[i] = models.MemberBalance{
			MemberID: m.MemberID, MemberName: m.MemberName, MemberIcon: m.MemberIcon,
		}
	}
	return out
}

func sumBalances(g *models.SharedGroupWithMembers, a, b []models.MemberBalance) []models.MemberBalance {
	am := make(map[int64]models.MemberBalance, len(a))
	for _, v := range a {
		am[v.MemberID] = v
	}
	bm := make(map[int64]models.MemberBalance, len(b))
	for _, v := range b {
		bm[v.MemberID] = v
	}
	out := make([]models.MemberBalance, len(g.Members))
	for i, m := range g.Members {
		av, bv := am[m.MemberID], bm[m.MemberID]
		tot := av.TotalPaid + bv.TotalPaid
		fair := av.FairShare + bv.FairShare
		out[i] = models.MemberBalance{
			MemberID:   m.MemberID,
			MemberName: m.MemberName,
			MemberIcon: m.MemberIcon,
			TotalPaid:  round2(tot),
			FairShare:  round2(fair),
			Balance:    round2(tot - fair),
		}
	}
	return out
}