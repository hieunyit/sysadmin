package handlers

type createUserDTO struct {
	Username             string `json:"username" validate:"required,min=3,max=64"`
	Email                string `json:"email" validate:"required,email"`
	NotificationEmail    string `json:"notification_email,omitempty" validate:"omitempty,email"`
	NotificationEmailAlt string `json:"notificationEmail,omitempty" validate:"omitempty,email"`

	FirstName    string `json:"first_name,omitempty"`
	FirstNameAlt string `json:"firstName,omitempty"`
	LastName     string `json:"last_name,omitempty"`
	LastNameAlt  string `json:"lastName,omitempty"`

	DisplayName       string `json:"display_name,omitempty"`
	DisplayNameAlt    string `json:"displayName,omitempty"`
	IdentitySource    string `json:"identity_source,omitempty" validate:"omitempty,oneof=local ldap"`
	IdentitySourceAlt string `json:"identitySource,omitempty" validate:"omitempty,oneof=local ldap"`

	EmailVerified    *bool `json:"email_verified,omitempty"`
	EmailVerifiedAlt *bool `json:"emailVerified,omitempty"`

	RequiredActions    []string `json:"required_actions,omitempty"`
	RequiredActionsAlt []string `json:"requiredActions,omitempty"`

	Attributes     map[string]string `json:"attributes"`
	Groups         []string          `json:"groups"`
	OnboardDate    string            `json:"onboard_date,omitempty"`
	OnboardDateAlt string            `json:"onboardDate,omitempty"`
	OnboardLegacy  string            `json:"onboard,omitempty"`
	WorkAddress    string            `json:"work_address,omitempty"`
	WorkAddressAlt string            `json:"workAddress,omitempty"`
	AddressLegacy  string            `json:"address,omitempty"`
	Enabled        bool              `json:"enabled"`

	Password string `json:"password" validate:"omitempty,min=8,max=255"`

	PasswordTemporary    *bool `json:"password_temporary,omitempty"`
	PasswordTemporaryAlt *bool `json:"passwordTemporary,omitempty"`
}

type updateUserDTO struct {
	Username *string `json:"username,omitempty" validate:"omitempty,min=3,max=64"`

	Email *string `json:"email,omitempty" validate:"omitempty,email"`

	FirstName    *string `json:"first_name,omitempty" validate:"omitempty,min=1,max=255"`
	FirstNameAlt *string `json:"firstName,omitempty" validate:"omitempty,min=1,max=255"`
	LastName     *string `json:"last_name,omitempty" validate:"omitempty,min=1,max=255"`
	LastNameAlt  *string `json:"lastName,omitempty" validate:"omitempty,min=1,max=255"`

	DisplayName    *string `json:"display_name,omitempty" validate:"omitempty,min=1,max=128"`
	DisplayNameAlt *string `json:"displayName,omitempty" validate:"omitempty,min=1,max=128"`

	Enabled *bool `json:"enabled,omitempty"`

	EmailVerified    *bool `json:"email_verified,omitempty"`
	EmailVerifiedAlt *bool `json:"emailVerified,omitempty"`

	RequiredActions    *[]string `json:"required_actions,omitempty"`
	RequiredActionsAlt *[]string `json:"requiredActions,omitempty"`

	Attributes *map[string]string `json:"attributes,omitempty"`
}

type createGroupDTO struct {
	Name string `json:"name" validate:"required,min=2,max=128"`
}

type updateGroupDTO struct {
	Name *string `json:"name" validate:"omitempty,min=2,max=128"`
}

type addMemberDTO struct {
	UserID string `json:"user_id" validate:"required"`
}

type accessRouteDTO struct {
	Target      *string `json:"target,omitempty"`
	Username    *string `json:"username,omitempty"`
	Groupname   *string `json:"groupname,omitempty"`
	Type        string  `json:"type,omitempty"`
	RouteType   string  `json:"route_type,omitempty"`
	Accept      *bool   `json:"accept,omitempty"`
	CIDR        *string `json:"cidr,omitempty"`
	ServiceSpec *string `json:"service_spec,omitempty"`
	Domain      *string `json:"domain,omitempty"`
	MatchType   string  `json:"match_type,omitempty"`
	Action      string  `json:"action,omitempty"`
	Position    *int    `json:"position,omitempty"`
	Comment     string  `json:"comment,omitempty"`
}

type accessListDTO struct {
	Items []accessRouteDTO `json:"items" validate:"required,min=1,dive"`
}

type createOpenVPNFromKeycloakDTO struct {
	UserID    string `json:"user_id,omitempty"`
	UserIDAlt string `json:"userId,omitempty"`
	Username  string `json:"username,omitempty"`
	VPNGroup  string `json:"vpn_group,omitempty"`
}
