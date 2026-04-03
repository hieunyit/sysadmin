package services

import "context"

type KeycloakUser struct {
	ID                string
	Username          string
	Email             string
	DisplayName       string
	FirstName         string
	LastName          string
	Enabled           bool
	EmailVerified     bool
	RequiredActions   []string
	Attributes        map[string]string
	Groups            []string
	Password          string
	PasswordTemporary bool
	IdentitySource    string
	LastVPNLoginAt    string
	LastVPNLoginLookupFailed bool
}

type KeycloakGroup struct {
	ID   string
	Name string
	Path string
}

type KeycloakUserLookup struct {
	Username                 string
	Email                    string
	DisplayName              string
	Enabled                  bool
	VPNExpireAt              string
	Groups                   []string
	LastVPNLoginAt           string
	LastVPNLoginLookupFailed bool
}

type KeycloakIdentityService interface {
	CreateUser(ctx context.Context, req KeycloakUser) (KeycloakUser, error)
	UpdateUser(ctx context.Context, id string, req KeycloakUser) (KeycloakUser, error)
	DeleteUser(ctx context.Context, id string) error
	GetUser(ctx context.Context, id string) (KeycloakUser, error)
	SearchUsers(ctx context.Context, q string, first, max int) ([]KeycloakUser, error)
	CreateGroup(ctx context.Context, req KeycloakGroup) (KeycloakGroup, error)
	UpdateGroup(ctx context.Context, id string, req KeycloakGroup) (KeycloakGroup, error)
	DeleteGroup(ctx context.Context, id string) error
	GetGroup(ctx context.Context, id string) (KeycloakGroup, error)
	SearchGroups(ctx context.Context, q string) ([]KeycloakGroup, error)
	ListGroupMembers(ctx context.Context, groupID string) ([]KeycloakUser, error)
	AddUserToGroup(ctx context.Context, userID, groupID string) error
	RemoveUserFromGroup(ctx context.Context, userID, groupID string) error
}
