package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	domainerr "backend/internal/domain/errors"
)

type AuditLog struct {
	ID         string
	Timestamp  time.Time
	Actor      string
	ActorType  string
	Action     string
	ActionType string
	Target     string
	TargetType string
	Source     string
	Status     string
	IPAddress  string
	Details    string
}

type AuditLogFilter struct {
	Limit      int
	Offset     int
	Search     string
	Source     string
	Status     string
	ActionType string
}

type DashboardSummary struct {
	TotalUsers     int
	ActiveSessions int
	VPNConnections int
	TotalVPNUsers  int
	RecentActivity []AuditLog
}

type RoleRef struct {
	ID   string
	Name string
}

type KeycloakUserView struct {
	ID               string
	Username         string
	Email            string
	FirstName        string
	LastName         string
	Enabled          bool
	EmailVerified    bool
	CreatedTimestamp int64
	Groups           []string
	Roles            []string
}

type KeycloakGroupView struct {
	ID        string
	Name      string
	Path      string
	SubGroups []KeycloakGroupView
}

type KeycloakRoleView struct {
	ID          string
	Name        string
	Description string
	Composite   bool
	ClientRole  bool
	ContainerID string
	UserCount   int
}

type KeycloakSessionView struct {
	ID         string
	Username   string
	UserID     string
	IPAddress  string
	Start      int64
	LastAccess int64
	Clients    map[string]string
}

type OpenVPNUserView struct {
	ID               string
	Username         string
	Email            string
	Enabled          bool
	CreatedAt        string
	LastLogin        *string
	Status           string
	AssignedIP       string
	Group            string
	PropAutologin    bool
	PropAdmin        bool
	PropDeny         bool
	PropAutogenerate bool
	AccessFrom       []string
	AccessTo         []string
}

type OpenVPNUserProps struct {
	PropAutologin    bool
	PropAdmin        bool
	PropDeny         bool
	PropAutogenerate bool
	Group            *string
	AccessFrom       []string
	AccessTo         []string
}

type OpenVPNGroupView struct {
	ID            string
	Name          string
	Description   string
	UserCount     int
	PropAutologin bool
	PropDeny      bool
	GroupSubnets  []string
	AccessFrom    []string
	AccessTo      []string
}

type OpenVPNGroupProps struct {
	PropAutologin bool
	PropDeny      bool
	GroupSubnets  []string
	AccessFrom    []string
	AccessTo      []string
}

type OpenVPNConnectionView struct {
	ID             string
	Username       string
	RealAddress    string
	VirtualAddress string
	BytesReceived  int64
	BytesSent      int64
	ConnectedSince string
	ClientID       string
	DisconnectedAt *string
}

type OpenVPNConfigView struct {
	ID            string
	Name          string
	Username      string
	CreatedAt     string
	ExpiresAt     *string
	DownloadCount int
	Status        string
}

type OpenVPNConfigDownload struct {
	Filename    string
	ContentType string
	Content     string
}

type KeycloakUserCreateInput struct {
	Username          string
	Email             string
	FirstName         string
	LastName          string
	Password          string
	Enabled           bool
	TemporaryPassword bool
}

type KeycloakUserUpdateInput struct {
	Email     *string
	FirstName *string
	LastName  *string
	Enabled   *bool
}

type OpenVPNUserCreateInput struct {
	Username      string
	Email         string
	Group         *string
	Enabled       bool
	PropAutologin bool
	PropAdmin     bool
}

type OpenVPNUserUpdateInput struct {
	Email   *string
	Enabled *bool
}

type OpenVPNGroupCreateInput struct {
	Name          string
	Description   string
	PropAutologin bool
}

type OpenVPNGroupUpdateInput struct {
	Description *string
}

type OpenVPNConfigCreateInput struct {
	Username string
	Name     string
}

type AdminPortalRepository interface {
	GetDashboardSummary(ctx context.Context, recentLimit int) (DashboardSummary, error)
	ListAuditLogs(ctx context.Context, filter AuditLogFilter) ([]AuditLog, int, error)
	InsertAuditLog(ctx context.Context, entry AuditLog) error

	ListKeycloakUsers(ctx context.Context, search string, first, max int) ([]KeycloakUserView, int, error)
	CreateKeycloakUser(ctx context.Context, input KeycloakUserCreateInput) (KeycloakUserView, error)
	UpdateKeycloakUser(ctx context.Context, id string, input KeycloakUserUpdateInput) (KeycloakUserView, error)
	DeleteKeycloakUser(ctx context.Context, id string) error
	SetKeycloakUserPassword(ctx context.Context, id, passwordHash string, temporary bool) error
	GetKeycloakUserRoles(ctx context.Context, id string) ([]KeycloakRoleView, error)
	AssignRolesToKeycloakUser(ctx context.Context, userID string, roles []RoleRef) error
	RemoveRolesFromKeycloakUser(ctx context.Context, userID string, roles []RoleRef) error
	GetKeycloakUserGroups(ctx context.Context, id string) ([]KeycloakGroupView, error)
	AddKeycloakUserToGroup(ctx context.Context, userID, groupID string) error
	RemoveKeycloakUserFromGroup(ctx context.Context, userID, groupID string) error

	ListKeycloakGroups(ctx context.Context) ([]KeycloakGroupView, error)
	CreateKeycloakGroup(ctx context.Context, name string, parentID *string) (KeycloakGroupView, error)
	UpdateKeycloakGroup(ctx context.Context, id, name string) (KeycloakGroupView, error)
	DeleteKeycloakGroup(ctx context.Context, id string) error
	GetKeycloakGroupMembers(ctx context.Context, id string) ([]KeycloakUserView, error)
	GetKeycloakGroupRoles(ctx context.Context, id string) ([]KeycloakRoleView, error)
	AssignRolesToKeycloakGroup(ctx context.Context, groupID string, roles []RoleRef) error
	RemoveRolesFromKeycloakGroup(ctx context.Context, groupID string, roles []RoleRef) error

	ListKeycloakRoles(ctx context.Context) ([]KeycloakRoleView, error)
	CreateKeycloakRole(ctx context.Context, name, description string) (KeycloakRoleView, error)
	DeleteKeycloakRole(ctx context.Context, name string) error

	ListKeycloakSessions(ctx context.Context) ([]KeycloakSessionView, error)
	LogoutKeycloakSession(ctx context.Context, sessionID string) error
	LogoutKeycloakUser(ctx context.Context, userID string) error
	LogoutAllKeycloakSessions(ctx context.Context) error

	ListOpenVPNUsers(ctx context.Context) ([]OpenVPNUserView, error)
	CreateOpenVPNUser(ctx context.Context, input OpenVPNUserCreateInput) (OpenVPNUserView, error)
	UpdateOpenVPNUser(ctx context.Context, username string, input OpenVPNUserUpdateInput) (OpenVPNUserView, error)
	DeleteOpenVPNUser(ctx context.Context, username string) error
	SetOpenVPNUserProps(ctx context.Context, username string, props OpenVPNUserProps) error
	GetOpenVPNUserProps(ctx context.Context, username string) (OpenVPNUserProps, error)
	GenerateOpenVPNUserMFA(ctx context.Context, username string) (string, error)

	ListOpenVPNGroups(ctx context.Context) ([]OpenVPNGroupView, error)
	CreateOpenVPNGroup(ctx context.Context, input OpenVPNGroupCreateInput) (OpenVPNGroupView, error)
	UpdateOpenVPNGroup(ctx context.Context, groupname string, input OpenVPNGroupUpdateInput) (OpenVPNGroupView, error)
	DeleteOpenVPNGroup(ctx context.Context, groupname string) error
	GetOpenVPNGroupProps(ctx context.Context, groupname string) (OpenVPNGroupProps, error)
	SetOpenVPNGroupProps(ctx context.Context, groupname string, props OpenVPNGroupProps) error
	GetOpenVPNGroupMembers(ctx context.Context, groupname string) ([]OpenVPNUserView, error)

	ListOpenVPNConnections(ctx context.Context, history bool) ([]OpenVPNConnectionView, error)
	DisconnectOpenVPNConnection(ctx context.Context, clientID string) error

	ListOpenVPNConfigs(ctx context.Context, username *string) ([]OpenVPNConfigView, error)
	CreateOpenVPNConfig(ctx context.Context, input OpenVPNConfigCreateInput) (OpenVPNConfigView, error)
	DownloadOpenVPNConfig(ctx context.Context, configID string) (OpenVPNConfigDownload, error)
	DeleteOpenVPNConfig(ctx context.Context, configID string) error
	RevokeOpenVPNConfig(ctx context.Context, configID string) error
	GetOpenVPNConfigQRCode(ctx context.Context, configID string) (string, error)
}

type AdminPortalService struct {
	repo AdminPortalRepository
}

func NewAdminPortalService(repo AdminPortalRepository) *AdminPortalService {
	return &AdminPortalService{repo: repo}
}

func (s *AdminPortalService) ensureConfigured() error {
	if s == nil || s.repo == nil {
		return domainerr.New(domainerr.CodePreconditionFail, "admin portal repository is not configured")
	}
	return nil
}

func (s *AdminPortalService) GetDashboardSummary(ctx context.Context) (DashboardSummary, error) {
	if err := s.ensureConfigured(); err != nil {
		return DashboardSummary{}, err
	}
	return s.repo.GetDashboardSummary(ctx, 4)
}

func (s *AdminPortalService) ListAuditLogs(ctx context.Context, filter AuditLogFilter) ([]AuditLog, int, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, 0, err
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	return s.repo.ListAuditLogs(ctx, filter)
}

func hashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func newPortalID(prefix string) string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buf[:]))
}

func (s *AdminPortalService) recordAudit(ctx context.Context, entry AuditLog) error {
	if s == nil || s.repo == nil {
		return nil
	}
	if strings.TrimSpace(entry.Actor) == "" {
		entry.Actor = "api-admin"
	}
	if strings.TrimSpace(entry.ActorType) == "" {
		entry.ActorType = "admin"
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = newPortalID("audit")
	}
	return s.repo.InsertAuditLog(ctx, entry)
}

func (s *AdminPortalService) ListKeycloakUsers(ctx context.Context, search string, first, max int) ([]KeycloakUserView, int, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, 0, err
	}
	if max <= 0 {
		max = 20
	}
	return s.repo.ListKeycloakUsers(ctx, search, first, max)
}

func (s *AdminPortalService) CreateKeycloakUser(ctx context.Context, input KeycloakUserCreateInput, actor string) (KeycloakUserView, error) {
	if err := s.ensureConfigured(); err != nil {
		return KeycloakUserView{}, err
	}
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	if input.Username == "" || input.Email == "" {
		return KeycloakUserView{}, domainerr.New(domainerr.CodeInvalidArgument, "username and email are required")
	}
	user, err := s.repo.CreateKeycloakUser(ctx, input)
	if err != nil {
		return KeycloakUserView{}, err
	}
	_ = s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "User Created",
		ActionType: "create",
		Target:     user.Username,
		TargetType: "user",
		Source:     "keycloak",
		Status:     "success",
		Details:    fmt.Sprintf("Created Keycloak user %s", user.Username),
	})
	return user, nil
}

func (s *AdminPortalService) UpdateKeycloakUser(ctx context.Context, id string, input KeycloakUserUpdateInput, actor string) (KeycloakUserView, error) {
	if err := s.ensureConfigured(); err != nil {
		return KeycloakUserView{}, err
	}
	user, err := s.repo.UpdateKeycloakUser(ctx, strings.TrimSpace(id), input)
	if err != nil {
		return KeycloakUserView{}, err
	}
	_ = s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "User Updated",
		ActionType: "update",
		Target:     user.Username,
		TargetType: "user",
		Source:     "keycloak",
		Status:     "success",
		Details:    fmt.Sprintf("Updated Keycloak user %s", user.Username),
	})
	return user, nil
}

func (s *AdminPortalService) DeleteKeycloakUser(ctx context.Context, id string, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.DeleteKeycloakUser(ctx, strings.TrimSpace(id)); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "User Deleted",
		ActionType: "delete",
		Target:     id,
		TargetType: "user",
		Source:     "keycloak",
		Status:     "success",
		Details:    "Deleted Keycloak user",
	})
}

func (s *AdminPortalService) ResetKeycloakUserPassword(ctx context.Context, id, password string, temporary bool, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if strings.TrimSpace(password) == "" {
		return domainerr.New(domainerr.CodeInvalidArgument, "password is required")
	}
	if err := s.repo.SetKeycloakUserPassword(ctx, strings.TrimSpace(id), hashPassword(password), temporary); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "Password Reset",
		ActionType: "update",
		Target:     id,
		TargetType: "user",
		Source:     "keycloak",
		Status:     "success",
		Details:    "Reset user password",
	})
}

func (s *AdminPortalService) GetKeycloakUserRoles(ctx context.Context, id string) ([]KeycloakRoleView, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	return s.repo.GetKeycloakUserRoles(ctx, strings.TrimSpace(id))
}

func (s *AdminPortalService) AssignRolesToKeycloakUser(ctx context.Context, userID string, roles []RoleRef, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.AssignRolesToKeycloakUser(ctx, strings.TrimSpace(userID), roles); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "Role Assigned",
		ActionType: "assign",
		Target:     userID,
		TargetType: "role",
		Source:     "keycloak",
		Status:     "success",
		Details:    fmt.Sprintf("Assigned %d role(s) to user", len(roles)),
	})
}

func (s *AdminPortalService) RemoveRolesFromKeycloakUser(ctx context.Context, userID string, roles []RoleRef, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.RemoveRolesFromKeycloakUser(ctx, strings.TrimSpace(userID), roles); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "Role Revoked",
		ActionType: "revoke",
		Target:     userID,
		TargetType: "role",
		Source:     "keycloak",
		Status:     "success",
		Details:    fmt.Sprintf("Removed %d role(s) from user", len(roles)),
	})
}

func (s *AdminPortalService) GetKeycloakUserGroups(ctx context.Context, id string) ([]KeycloakGroupView, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	return s.repo.GetKeycloakUserGroups(ctx, strings.TrimSpace(id))
}

func (s *AdminPortalService) AddKeycloakUserToGroup(ctx context.Context, userID, groupID, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.AddKeycloakUserToGroup(ctx, strings.TrimSpace(userID), strings.TrimSpace(groupID)); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "User Added to Group",
		ActionType: "assign",
		Target:     userID,
		TargetType: "group",
		Source:     "keycloak",
		Status:     "success",
		Details:    fmt.Sprintf("Added user %s to group %s", userID, groupID),
	})
}

func (s *AdminPortalService) RemoveKeycloakUserFromGroup(ctx context.Context, userID, groupID, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.RemoveKeycloakUserFromGroup(ctx, strings.TrimSpace(userID), strings.TrimSpace(groupID)); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "User Removed from Group",
		ActionType: "revoke",
		Target:     userID,
		TargetType: "group",
		Source:     "keycloak",
		Status:     "success",
		Details:    fmt.Sprintf("Removed user %s from group %s", userID, groupID),
	})
}

func (s *AdminPortalService) LogoutKeycloakUser(ctx context.Context, userID, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.LogoutKeycloakUser(ctx, strings.TrimSpace(userID)); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "User Sessions Revoked",
		ActionType: "revoke",
		Target:     userID,
		TargetType: "session",
		Source:     "keycloak",
		Status:     "success",
		Details:    "Logged out all user sessions",
	})
}

func (s *AdminPortalService) SendKeycloakVerifyEmail(ctx context.Context, userID, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "Verification Email Sent",
		ActionType: "update",
		Target:     userID,
		TargetType: "user",
		Source:     "keycloak",
		Status:     "success",
		Details:    "Triggered verification email",
	})
}

func (s *AdminPortalService) ListKeycloakGroups(ctx context.Context) ([]KeycloakGroupView, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	return s.repo.ListKeycloakGroups(ctx)
}

func (s *AdminPortalService) CreateKeycloakGroup(ctx context.Context, name string, parentID *string, actor string) (KeycloakGroupView, error) {
	if err := s.ensureConfigured(); err != nil {
		return KeycloakGroupView{}, err
	}
	group, err := s.repo.CreateKeycloakGroup(ctx, strings.TrimSpace(name), parentID)
	if err != nil {
		return KeycloakGroupView{}, err
	}
	_ = s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "Group Created",
		ActionType: "create",
		Target:     group.Name,
		TargetType: "group",
		Source:     "keycloak",
		Status:     "success",
		Details:    fmt.Sprintf("Created group %s", group.Path),
	})
	return group, nil
}

func (s *AdminPortalService) UpdateKeycloakGroup(ctx context.Context, id, name, actor string) (KeycloakGroupView, error) {
	if err := s.ensureConfigured(); err != nil {
		return KeycloakGroupView{}, err
	}
	group, err := s.repo.UpdateKeycloakGroup(ctx, strings.TrimSpace(id), strings.TrimSpace(name))
	if err != nil {
		return KeycloakGroupView{}, err
	}
	_ = s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "Group Updated",
		ActionType: "update",
		Target:     group.Name,
		TargetType: "group",
		Source:     "keycloak",
		Status:     "success",
		Details:    fmt.Sprintf("Updated group %s", group.Path),
	})
	return group, nil
}

func (s *AdminPortalService) DeleteKeycloakGroup(ctx context.Context, id, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.DeleteKeycloakGroup(ctx, strings.TrimSpace(id)); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "Group Deleted",
		ActionType: "delete",
		Target:     id,
		TargetType: "group",
		Source:     "keycloak",
		Status:     "success",
		Details:    "Deleted group hierarchy",
	})
}

func (s *AdminPortalService) GetKeycloakGroupMembers(ctx context.Context, id string) ([]KeycloakUserView, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	return s.repo.GetKeycloakGroupMembers(ctx, strings.TrimSpace(id))
}

func (s *AdminPortalService) GetKeycloakGroupRoles(ctx context.Context, id string) ([]KeycloakRoleView, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	return s.repo.GetKeycloakGroupRoles(ctx, strings.TrimSpace(id))
}

func (s *AdminPortalService) AssignRolesToKeycloakGroup(ctx context.Context, groupID string, roles []RoleRef, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.AssignRolesToKeycloakGroup(ctx, strings.TrimSpace(groupID), roles); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "Group Role Assigned",
		ActionType: "assign",
		Target:     groupID,
		TargetType: "group",
		Source:     "keycloak",
		Status:     "success",
		Details:    fmt.Sprintf("Assigned %d role(s) to group", len(roles)),
	})
}

func (s *AdminPortalService) RemoveRolesFromKeycloakGroup(ctx context.Context, groupID string, roles []RoleRef, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.RemoveRolesFromKeycloakGroup(ctx, strings.TrimSpace(groupID), roles); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "Group Role Removed",
		ActionType: "revoke",
		Target:     groupID,
		TargetType: "group",
		Source:     "keycloak",
		Status:     "success",
		Details:    fmt.Sprintf("Removed %d role(s) from group", len(roles)),
	})
}

func (s *AdminPortalService) ListKeycloakRoles(ctx context.Context) ([]KeycloakRoleView, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	return s.repo.ListKeycloakRoles(ctx)
}

func (s *AdminPortalService) CreateKeycloakRole(ctx context.Context, name, description, actor string) (KeycloakRoleView, error) {
	if err := s.ensureConfigured(); err != nil {
		return KeycloakRoleView{}, err
	}
	role, err := s.repo.CreateKeycloakRole(ctx, strings.TrimSpace(name), strings.TrimSpace(description))
	if err != nil {
		return KeycloakRoleView{}, err
	}
	_ = s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "Role Created",
		ActionType: "create",
		Target:     role.Name,
		TargetType: "role",
		Source:     "keycloak",
		Status:     "success",
		Details:    fmt.Sprintf("Created role %s", role.Name),
	})
	return role, nil
}

func (s *AdminPortalService) DeleteKeycloakRole(ctx context.Context, name, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.DeleteKeycloakRole(ctx, strings.TrimSpace(name)); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "Role Deleted",
		ActionType: "delete",
		Target:     name,
		TargetType: "role",
		Source:     "keycloak",
		Status:     "success",
		Details:    "Deleted role",
	})
}

func (s *AdminPortalService) ListKeycloakSessions(ctx context.Context) ([]KeycloakSessionView, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	return s.repo.ListKeycloakSessions(ctx)
}

func (s *AdminPortalService) LogoutKeycloakSession(ctx context.Context, sessionID, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.LogoutKeycloakSession(ctx, strings.TrimSpace(sessionID)); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "Session Logged Out",
		ActionType: "logout",
		Target:     sessionID,
		TargetType: "session",
		Source:     "keycloak",
		Status:     "success",
		Details:    "Logged out a single session",
	})
}

func (s *AdminPortalService) LogoutAllKeycloakSessions(ctx context.Context, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.LogoutAllKeycloakSessions(ctx); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "All Sessions Logged Out",
		ActionType: "logout",
		Target:     "realm",
		TargetType: "session",
		Source:     "keycloak",
		Status:     "success",
		Details:    "Logged out all sessions",
	})
}

func (s *AdminPortalService) ListOpenVPNUsers(ctx context.Context) ([]OpenVPNUserView, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	return s.repo.ListOpenVPNUsers(ctx)
}

func (s *AdminPortalService) CreateOpenVPNUser(ctx context.Context, input OpenVPNUserCreateInput, actor string) (OpenVPNUserView, error) {
	if err := s.ensureConfigured(); err != nil {
		return OpenVPNUserView{}, err
	}
	user, err := s.repo.CreateOpenVPNUser(ctx, input)
	if err != nil {
		return OpenVPNUserView{}, err
	}
	_ = s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "VPN User Created",
		ActionType: "create",
		Target:     user.Username,
		TargetType: "vpn",
		Source:     "openvpn",
		Status:     "success",
		Details:    fmt.Sprintf("Created VPN user %s", user.Username),
	})
	return user, nil
}

func (s *AdminPortalService) UpdateOpenVPNUser(ctx context.Context, username string, input OpenVPNUserUpdateInput, actor string) (OpenVPNUserView, error) {
	if err := s.ensureConfigured(); err != nil {
		return OpenVPNUserView{}, err
	}
	user, err := s.repo.UpdateOpenVPNUser(ctx, strings.TrimSpace(username), input)
	if err != nil {
		return OpenVPNUserView{}, err
	}
	_ = s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "VPN User Updated",
		ActionType: "update",
		Target:     user.Username,
		TargetType: "vpn",
		Source:     "openvpn",
		Status:     "success",
		Details:    "Updated VPN user",
	})
	return user, nil
}

func (s *AdminPortalService) DeleteOpenVPNUser(ctx context.Context, username, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.DeleteOpenVPNUser(ctx, strings.TrimSpace(username)); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "VPN User Deleted",
		ActionType: "delete",
		Target:     username,
		TargetType: "vpn",
		Source:     "openvpn",
		Status:     "success",
		Details:    "Deleted VPN user",
	})
}

func (s *AdminPortalService) EnableOpenVPNUser(ctx context.Context, username, actor string) error {
	props, err := s.GetOpenVPNUserProps(ctx, username)
	if err != nil {
		return err
	}
	props.PropDeny = false
	return s.SetOpenVPNUserProps(ctx, username, props, actor, "VPN User Enabled", "enable")
}

func (s *AdminPortalService) DisableOpenVPNUser(ctx context.Context, username, actor string) error {
	props, err := s.GetOpenVPNUserProps(ctx, username)
	if err != nil {
		return err
	}
	props.PropDeny = true
	return s.SetOpenVPNUserProps(ctx, username, props, actor, "VPN User Disabled", "disable")
}

func (s *AdminPortalService) GetOpenVPNUserProps(ctx context.Context, username string) (OpenVPNUserProps, error) {
	if err := s.ensureConfigured(); err != nil {
		return OpenVPNUserProps{}, err
	}
	return s.repo.GetOpenVPNUserProps(ctx, strings.TrimSpace(username))
}

func (s *AdminPortalService) SetOpenVPNUserProps(ctx context.Context, username string, props OpenVPNUserProps, actor, action, actionType string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if action == "" {
		action = "VPN User Properties Updated"
	}
	if actionType == "" {
		actionType = "update"
	}
	if err := s.repo.SetOpenVPNUserProps(ctx, strings.TrimSpace(username), props); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     action,
		ActionType: actionType,
		Target:     username,
		TargetType: "vpn",
		Source:     "openvpn",
		Status:     "success",
		Details:    "Updated VPN user properties",
	})
}

func (s *AdminPortalService) GenerateOpenVPNUserMFA(ctx context.Context, username, actor string) (string, error) {
	if err := s.ensureConfigured(); err != nil {
		return "", err
	}
	secret, err := s.repo.GenerateOpenVPNUserMFA(ctx, strings.TrimSpace(username))
	if err != nil {
		return "", err
	}
	_ = s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "VPN MFA Secret Generated",
		ActionType: "update",
		Target:     username,
		TargetType: "vpn",
		Source:     "openvpn",
		Status:     "success",
		Details:    "Generated a new MFA secret",
	})
	return secret, nil
}

func (s *AdminPortalService) ListOpenVPNGroups(ctx context.Context) ([]OpenVPNGroupView, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	return s.repo.ListOpenVPNGroups(ctx)
}

func (s *AdminPortalService) CreateOpenVPNGroup(ctx context.Context, input OpenVPNGroupCreateInput, actor string) (OpenVPNGroupView, error) {
	if err := s.ensureConfigured(); err != nil {
		return OpenVPNGroupView{}, err
	}
	group, err := s.repo.CreateOpenVPNGroup(ctx, input)
	if err != nil {
		return OpenVPNGroupView{}, err
	}
	_ = s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "VPN Group Created",
		ActionType: "create",
		Target:     group.Name,
		TargetType: "group",
		Source:     "openvpn",
		Status:     "success",
		Details:    "Created VPN group",
	})
	return group, nil
}

func (s *AdminPortalService) UpdateOpenVPNGroup(ctx context.Context, groupname string, input OpenVPNGroupUpdateInput, actor string) (OpenVPNGroupView, error) {
	if err := s.ensureConfigured(); err != nil {
		return OpenVPNGroupView{}, err
	}
	group, err := s.repo.UpdateOpenVPNGroup(ctx, strings.TrimSpace(groupname), input)
	if err != nil {
		return OpenVPNGroupView{}, err
	}
	_ = s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "VPN Group Updated",
		ActionType: "update",
		Target:     group.Name,
		TargetType: "group",
		Source:     "openvpn",
		Status:     "success",
		Details:    "Updated VPN group",
	})
	return group, nil
}

func (s *AdminPortalService) DeleteOpenVPNGroup(ctx context.Context, groupname, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.DeleteOpenVPNGroup(ctx, strings.TrimSpace(groupname)); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "VPN Group Deleted",
		ActionType: "delete",
		Target:     groupname,
		TargetType: "group",
		Source:     "openvpn",
		Status:     "success",
		Details:    "Deleted VPN group",
	})
}

func (s *AdminPortalService) GetOpenVPNGroupProps(ctx context.Context, groupname string) (OpenVPNGroupProps, error) {
	if err := s.ensureConfigured(); err != nil {
		return OpenVPNGroupProps{}, err
	}
	return s.repo.GetOpenVPNGroupProps(ctx, strings.TrimSpace(groupname))
}

func (s *AdminPortalService) SetOpenVPNGroupProps(ctx context.Context, groupname string, props OpenVPNGroupProps, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.SetOpenVPNGroupProps(ctx, strings.TrimSpace(groupname), props); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "VPN Group Properties Updated",
		ActionType: "update",
		Target:     groupname,
		TargetType: "group",
		Source:     "openvpn",
		Status:     "success",
		Details:    "Updated VPN group properties",
	})
}

func (s *AdminPortalService) GetOpenVPNGroupMembers(ctx context.Context, groupname string) ([]OpenVPNUserView, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	return s.repo.GetOpenVPNGroupMembers(ctx, strings.TrimSpace(groupname))
}

func (s *AdminPortalService) ListOpenVPNConnections(ctx context.Context, history bool) ([]OpenVPNConnectionView, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	return s.repo.ListOpenVPNConnections(ctx, history)
}

func (s *AdminPortalService) DisconnectOpenVPNConnection(ctx context.Context, clientID, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.DisconnectOpenVPNConnection(ctx, strings.TrimSpace(clientID)); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "VPN Session Ended",
		ActionType: "logout",
		Target:     clientID,
		TargetType: "vpn",
		Source:     "openvpn",
		Status:     "success",
		Details:    "Disconnected VPN connection",
	})
}

func (s *AdminPortalService) ListOpenVPNConfigs(ctx context.Context, username *string) ([]OpenVPNConfigView, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	return s.repo.ListOpenVPNConfigs(ctx, username)
}

func (s *AdminPortalService) CreateOpenVPNConfig(ctx context.Context, input OpenVPNConfigCreateInput, actor string) (OpenVPNConfigView, error) {
	if err := s.ensureConfigured(); err != nil {
		return OpenVPNConfigView{}, err
	}
	config, err := s.repo.CreateOpenVPNConfig(ctx, input)
	if err != nil {
		return OpenVPNConfigView{}, err
	}
	_ = s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "VPN Config Generated",
		ActionType: "create",
		Target:     config.Name,
		TargetType: "vpn",
		Source:     "openvpn",
		Status:     "success",
		Details:    fmt.Sprintf("Generated config for %s", config.Username),
	})
	return config, nil
}

func (s *AdminPortalService) DownloadOpenVPNConfig(ctx context.Context, configID, actor string) (OpenVPNConfigDownload, error) {
	if err := s.ensureConfigured(); err != nil {
		return OpenVPNConfigDownload{}, err
	}
	download, err := s.repo.DownloadOpenVPNConfig(ctx, strings.TrimSpace(configID))
	if err != nil {
		return OpenVPNConfigDownload{}, err
	}
	_ = s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "VPN Config Downloaded",
		ActionType: "update",
		Target:     configID,
		TargetType: "vpn",
		Source:     "openvpn",
		Status:     "success",
		Details:    "Downloaded VPN configuration",
	})
	return download, nil
}

func (s *AdminPortalService) DeleteOpenVPNConfig(ctx context.Context, configID, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.DeleteOpenVPNConfig(ctx, strings.TrimSpace(configID)); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "VPN Config Deleted",
		ActionType: "delete",
		Target:     configID,
		TargetType: "vpn",
		Source:     "openvpn",
		Status:     "success",
		Details:    "Deleted VPN configuration",
	})
}

func (s *AdminPortalService) RevokeOpenVPNConfig(ctx context.Context, configID, actor string) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := s.repo.RevokeOpenVPNConfig(ctx, strings.TrimSpace(configID)); err != nil {
		return err
	}
	return s.recordAudit(ctx, AuditLog{
		Actor:      actor,
		ActorType:  "admin",
		Action:     "VPN Config Revoked",
		ActionType: "revoke",
		Target:     configID,
		TargetType: "vpn",
		Source:     "openvpn",
		Status:     "success",
		Details:    "Revoked VPN configuration",
	})
}

func (s *AdminPortalService) GetOpenVPNConfigQRCode(ctx context.Context, configID string) (string, error) {
	if err := s.ensureConfigured(); err != nil {
		return "", err
	}
	return s.repo.GetOpenVPNConfigQRCode(ctx, strings.TrimSpace(configID))
}
