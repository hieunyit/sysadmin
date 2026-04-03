package commands

type CreateUser struct {
	Username          string            `json:"username" validate:"required,min=3,max=64"`
	Email             string            `json:"email" validate:"required,email"`
	NotificationEmail string            `json:"notification_email,omitempty" validate:"omitempty,email"`
	FirstName         string            `json:"first_name" validate:"required,min=1,max=255"`
	LastName          string            `json:"last_name" validate:"required,min=1,max=255"`
	DisplayName       string            `json:"display_name,omitempty" validate:"omitempty,min=1,max=255"`
	IdentitySource    string            `json:"identity_source,omitempty" validate:"omitempty,oneof=local ldap"`
	EmailVerified     bool              `json:"email_verified"`
	RequiredActions   []string          `json:"required_actions"`
	Attributes        map[string]string `json:"attributes"`
	Groups            []string          `json:"groups"`
	Enabled           bool              `json:"enabled"`
	Password          string            `json:"password" validate:"omitempty,min=8,max=255"`
	PasswordTemporary bool              `json:"password_temporary"`
	Actor             string            `json:"-"`
}

type UpdateUser struct {
	Username        *string            `json:"username" validate:"omitempty,min=3,max=64"`
	Email           *string            `json:"email" validate:"omitempty,email"`
	FirstName       *string            `json:"first_name" validate:"omitempty,min=1,max=255"`
	LastName        *string            `json:"last_name" validate:"omitempty,min=1,max=255"`
	DisplayName     *string            `json:"display_name" validate:"omitempty,min=1,max=128"`
	Enabled         *bool              `json:"enabled"`
	EmailVerified   *bool              `json:"email_verified"`
	RequiredActions *[]string          `json:"required_actions"`
	Attributes      *map[string]string `json:"attributes"`
	Actor           string             `json:"-"`
}
