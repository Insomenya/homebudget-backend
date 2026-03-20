package models

type Account struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Currency       string  `json:"currency"`
	InitialBalance float64 `json:"initial_balance"`
	MemberID       int64   `json:"member_id"`
	IsArchived     bool    `json:"is_archived"`
	IsHidden       bool    `json:"is_hidden"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type CreateAccountInput struct {
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Currency       string  `json:"currency"`
	InitialBalance float64 `json:"initial_balance"`
	MemberID       int64   `json:"member_id"`
	IsHidden       bool    `json:"is_hidden"`
}

type UpdateAccountInput struct {
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	Currency       string  `json:"currency"`
	InitialBalance float64 `json:"initial_balance"`
	MemberID       int64   `json:"member_id"`
	IsArchived     bool    `json:"is_archived"`
}

func (in *CreateAccountInput) Validate() string {
	if in.Name == "" {
		return "name is required"
	}
	if in.MemberID <= 0 {
		return "member_id is required"
	}
	if in.Type == "" {
		in.Type = "cash"
	}
	if in.Currency == "" {
		in.Currency = "RUB"
	}
	return ""
}

func (in *UpdateAccountInput) Validate() string {
	if in.Name == "" {
		return "name is required"
	}
	if in.MemberID <= 0 {
		return "member_id is required"
	}
	if in.Currency == "" {
		in.Currency = "RUB"
	}
	return ""
}