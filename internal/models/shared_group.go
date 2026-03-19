package models

import (
	"math"
	"sort"
)

// ── Сущности ────────────────────────────────────────────────────

type SharedGroup struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Icon       string `json:"icon"`
	IsArchived bool   `json:"is_archived"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type SharedGroupMember struct {
	ID               int64  `json:"id"`
	GroupID          int64  `json:"group_id"`
	MemberID         int64  `json:"member_id"`
	MemberName       string `json:"member_name"`
	MemberIcon       string `json:"member_icon"`
	ShareNumerator   int    `json:"share_numerator"`
	ShareDenominator int    `json:"share_denominator"`
}

func (m *SharedGroupMember) ShareFloat() float64 {
	if m.ShareDenominator == 0 {
		return 0
	}
	return float64(m.ShareNumerator) / float64(m.ShareDenominator)
}

type SharedGroupWithMembers struct {
	SharedGroup
	Members []SharedGroupMember `json:"members"`
}

// ── Ввод ────────────────────────────────────────────────────────

type SharedGroupMemberInput struct {
	MemberID         int64 `json:"member_id"`
	ShareNumerator   int   `json:"share_numerator"`
	ShareDenominator int   `json:"share_denominator"`
}

type CreateSharedGroupInput struct {
	Name    string                   `json:"name"`
	Icon    string                   `json:"icon"`
	Members []SharedGroupMemberInput `json:"members"`
}

type UpdateSharedGroupInput struct {
	Name       string                   `json:"name"`
	Icon       string                   `json:"icon"`
	IsArchived bool                     `json:"is_archived"`
	Members    []SharedGroupMemberInput `json:"members"`
}

func validateMembers(members []SharedGroupMemberInput) string {
	if len(members) < 2 {
		return "at least 2 members required"
	}
	seen := make(map[int64]bool)
	var sum float64
	for _, m := range members {
		if m.MemberID <= 0 {
			return "invalid member_id"
		}
		if seen[m.MemberID] {
			return "duplicate member_id"
		}
		seen[m.MemberID] = true
		if m.ShareNumerator <= 0 || m.ShareDenominator <= 0 {
			return "shares must be positive"
		}
		sum += float64(m.ShareNumerator) / float64(m.ShareDenominator)
	}
	if math.Abs(sum-1.0) > 0.001 {
		return "shares must sum to 1"
	}
	return ""
}

func (in *CreateSharedGroupInput) Validate() string {
	if in.Name == "" {
		return "name is required"
	}
	return validateMembers(in.Members)
}

func (in *UpdateSharedGroupInput) Validate() string {
	if in.Name == "" {
		return "name is required"
	}
	return validateMembers(in.Members)
}

// ── Балансы и долги ─────────────────────────────────────────────

type MemberBalance struct {
	MemberID   int64   `json:"member_id"`
	MemberName string  `json:"member_name"`
	MemberIcon string  `json:"member_icon"`
	TotalPaid  float64 `json:"total_paid"`
	FairShare  float64 `json:"fair_share"`
	Balance    float64 `json:"balance"`
	Percentage float64 `json:"percentage"`
}

type Debt struct {
	FromMemberID   int64   `json:"from_member_id"`
	FromMemberName string  `json:"from_member_name"`
	ToMemberID     int64   `json:"to_member_id"`
	ToMemberName   string  `json:"to_member_name"`
	Amount         float64 `json:"amount"`
}

type Settlement struct {
	Group         SharedGroupWithMembers `json:"group"`
	TotalExpenses float64                `json:"total_expenses"`
	Balances      []MemberBalance        `json:"balances"`
	Debts         []Debt                 `json:"debts"`
}

func ComputeDebts(balances []MemberBalance) []Debt {
	type entry struct {
		id   int64
		name string
		amt  float64
	}

	var cred, debt []entry
	for _, b := range balances {
		if b.Balance > 0.01 {
			cred = append(cred, entry{b.MemberID, b.MemberName, b.Balance})
		} else if b.Balance < -0.01 {
			debt = append(debt, entry{b.MemberID, b.MemberName, -b.Balance})
		}
	}

	sort.Slice(cred, func(i, j int) bool { return cred[i].amt > cred[j].amt })
	sort.Slice(debt, func(i, j int) bool { return debt[i].amt > debt[j].amt })

	var out []Debt
	i, j := 0, 0
	for i < len(cred) && j < len(debt) {
		a := math.Min(cred[i].amt, debt[j].amt)
		out = append(out, Debt{
			FromMemberID: debt[j].id, FromMemberName: debt[j].name,
			ToMemberID: cred[i].id, ToMemberName: cred[i].name,
			Amount: math.Round(a*100) / 100,
		})
		cred[i].amt -= a
		debt[j].amt -= a
		if cred[i].amt < 0.01 {
			i++
		}
		if debt[j].amt < 0.01 {
			j++
		}
	}
	return out
}