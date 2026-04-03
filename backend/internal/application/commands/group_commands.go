package commands

type CreateGroup struct {
	Name  string `json:"name" validate:"required,min=2,max=128"`
	Actor string `json:"-"`
}

type UpdateGroup struct {
	Name  *string `json:"name" validate:"omitempty,min=2,max=128"`
	Actor string  `json:"-"`
}

type AddGroupMember struct {
	UserID string `json:"user_id" validate:"required"`
	Actor  string `json:"-"`
}
