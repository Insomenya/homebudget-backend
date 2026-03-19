package models

type Member struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Icon       string `json:"icon"`
	IsArchived bool   `json:"is_archived"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type CreateMemberInput struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type UpdateMemberInput struct {
	Name       string `json:"name"`
	Icon       string `json:"icon"`
	IsArchived bool   `json:"is_archived"`
}

func (in *CreateMemberInput) Validate() string {
	if in.Name == "" {
		return "name is required"
	}
	return ""
}

func (in *UpdateMemberInput) Validate() string {
	if in.Name == "" {
		return "name is required"
	}
	return ""
}