package models

const (
	TxTypeExpense  = "expense"
	TxTypeIncome   = "income"
	TxTypeTransfer = "transfer"
)

type Transaction struct {
	ID              int64   `json:"id"`
	Date            string  `json:"date"`
	Amount          float64 `json:"amount"`
	Description     string  `json:"description"`
	Type            string  `json:"type"`
	AccountID       *int64  `json:"account_id"`
	ToAccountID     *int64  `json:"to_account_id"`
	CategoryID      *int64  `json:"category_id"`
	SharedGroupID   *int64  `json:"shared_group_id"`
	PaidByMemberID  *int64  `json:"paid_by_member_id"`
	LoanID          *int64  `json:"loan_id"`
	ReminderID      *int64  `json:"reminder_id"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type CreateTransactionInput struct {
	Date            string  `json:"date"`
	Amount          float64 `json:"amount"`
	Description     string  `json:"description"`
	Type            string  `json:"type"`
	AccountID       *int64  `json:"account_id"`
	ToAccountID     *int64  `json:"to_account_id"`
	CategoryID      *int64  `json:"category_id"`
	SharedGroupID   *int64  `json:"shared_group_id"`
	PaidByMemberID  *int64  `json:"paid_by_member_id"`
	LoanID          *int64  `json:"loan_id"`
	ReminderID      *int64  `json:"reminder_id"`
}

type UpdateTransactionInput = CreateTransactionInput

func (in *CreateTransactionInput) Validate() string {
	if in.Date == "" {
		return "date is required"
	}
	if in.Amount <= 0 {
		return "amount must be positive"
	}

	switch in.Type {
	case TxTypeExpense, TxTypeIncome, TxTypeTransfer:
	case "":
		in.Type = TxTypeExpense
	default:
		return "type must be expense, income, or transfer"
	}

	switch in.Type {
	case TxTypeExpense:
		if in.CategoryID == nil {
			return "category_id required for expense"
		}
	case TxTypeIncome:
		if in.AccountID == nil {
			return "account_id required for income"
		}
		if in.CategoryID == nil {
			return "category_id required for income"
		}
	case TxTypeTransfer:
		if in.AccountID == nil || in.ToAccountID == nil {
			return "account_id and to_account_id required for transfer"
		}
	}

	if in.SharedGroupID != nil && in.PaidByMemberID == nil {
		return "paid_by_member_id required for shared expense"
	}
	if in.PaidByMemberID != nil && in.SharedGroupID == nil {
		return "shared_group_id required when paid_by_member_id is set"
	}

	return ""
}

type TransactionFilter struct {
	DateFrom       string
	DateTo         string
	Search         string
	Type           string
	AccountID      *int64
	CategoryID     *int64
	SharedGroupID  *int64
	PaidByMemberID *int64
	LoanID         *int64
	IsShared       *bool
	Page           int
	Limit          int
	SortBy         string
	SortDir        string
}

type TransactionList struct {
	Items []Transaction `json:"items"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
	Pages int           `json:"pages"`
}