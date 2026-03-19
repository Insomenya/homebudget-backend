package models

type LookupValue struct {
	ID        int64  `json:"id"`
	GroupName string `json:"group_name"`
	Value     string `json:"value"`
	Label     string `json:"label"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

type CreateLookupInput struct {
	GroupName string `json:"group_name"`
	Label     string `json:"label"`
}

type UpdateLookupInput struct {
	Label    string `json:"label"`
	IsActive bool   `json:"is_active"`
}

func (in *CreateLookupInput) Validate() string {
	if in.GroupName == "" {
		return "group_name is required"
	}
	if in.Label == "" {
		return "label is required"
	}
	return ""
}

type Meta struct {
	AccountTypes     []LookupValue `json:"account_types"`
	Currencies       []LookupValue `json:"currencies"`
	TransactionTypes []LookupValue `json:"transaction_types"`
	CategoryTypes    []LookupValue `json:"category_types"`
	RecurrenceTypes  []LookupValue `json:"recurrence_types"`
}