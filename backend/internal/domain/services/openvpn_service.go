package services

import "context"

type OpenVPNRuleset struct {
	ID        int64
	Name      string
	Comment   string
	Owner     string
	OwnerType string
	Position  int
}

type OpenVPNRule struct {
	ID        *int64
	RulesetID int64
	Type      string
	MatchType string
	MatchData string
	Action    string
	Position  int
	Comment   string
}

type OpenVPNUserListQuery struct {
	Limit  int
	Offset int
	Search string
}

type OpenVPNGroupListQuery struct {
	Limit            int
	Offset           int
	Search           string
	EnumerateMembers bool
}

type OpenVPNAccessListQuery struct {
	Username    string
	Groupname   string
	SubjectType string // user|group
}

type OpenVPNAccessEntryInput struct {
	Username    *string
	Groupname   *string
	Target      *string
	Type        string
	RouteType   string
	Accept      *bool
	CIDR        *string
	ServiceSpec *string
	Domain      *string
	MatchType   string
	Action      string
	Position    *int
	Comment     string
}

type OpenVPNUserExportRow struct {
	Username        string `json:"username"`
	OpenVPNGroup    string `json:"group"`
	AuthMethod      string `json:"auth_method"`
	Deny            string `json:"deny"`
	PasswordDefined string `json:"password_defined"`
	MFAStatus       string `json:"mfa_status"`
	Email           string `json:"email,omitempty"`
	DisplayName     string `json:"display_name,omitempty"`
	Enabled         bool   `json:"enabled"`
	VPNExpireAt     string `json:"vpn_expire_at,omitempty"`
	LastVPNLoginAt  string `json:"last_vpn_login_at,omitempty"`
}

type OpenVPNRuleListQuery struct {
	RulesetIDs []int64
	Owner      string
	Type       string
	MatchType  string
	MatchData  string
}

type SubjectRulesetRef struct {
	RulesetID int64
	Position  int
}

type AccessRouteItem struct {
	Username    *string
	Groupname   *string
	Type        string // access_to_ipv4/access_from_ipv4...
	RouteType   string // route/nat/user/group/all
	Accept      bool
	CIDR        *string
	ServiceSpec *string
}

type OpenVPNPolicyService interface {
	AddRuleset(ctx context.Context, name, comment string) (int64, error)
	UpdateRuleset(ctx context.Context, id int64, name, comment string) error
	DeleteRulesets(ctx context.Context, ids []int64) error
	ListRulesets(ctx context.Context, owner, nameFilter string) ([]OpenVPNRuleset, error)
	ModifyRules(ctx context.Context, addOrUpdate []OpenVPNRule, deleteIDs []int64) ([]int64, error)
	ModifyUserRulesetMapping(ctx context.Context, add map[string][]SubjectRulesetRef, del map[string][]int64) error
	SetAccessList(ctx context.Context, items []AccessRouteItem) error
	AppendAccessList(ctx context.Context, items []AccessRouteItem) error
	RemoveAccessList(ctx context.Context, items []AccessRouteItem) error
	ListUsers(ctx context.Context, q OpenVPNUserListQuery) (map[string]any, error)
	ListGroups(ctx context.Context, q OpenVPNGroupListQuery) (map[string]any, error)
	ListAccessLists(ctx context.Context, q OpenVPNAccessListQuery) (map[string]any, error)
	ListRules(ctx context.Context, q OpenVPNRuleListQuery) (map[string]any, error)
	EnsureUser(ctx context.Context, username string) error
	EnsureGroup(ctx context.Context, groupname string) error
	SetUserGroup(ctx context.Context, username, groupname string) error
	DeleteUser(ctx context.Context, username string) error
	DeleteGroup(ctx context.Context, groupname string) error
}
