package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"backend/internal/application/commands"
	domainerr "backend/internal/domain/errors"
	"backend/internal/domain/services"
)

func TestValidateCreateUserBusinessRules_LocalNormalizesAndForcesTemporary(t *testing.T) {
	t.Parallel()

	cmd := &commands.CreateUser{
		Username:          "  local.user  ",
		Email:             " local.user@example.com ",
		FirstName:         " Local ",
		LastName:          " User ",
		IdentitySource:    " LOCAL ",
		PasswordTemporary: false,
		Groups:            []string{"  /vpn/local  ", "   "},
		RequiredActions:   []string{" UPDATE_PASSWORD ", " "},
		Attributes: map[string]string{
			"fullName":    " Local User ",
			"userType":    " partner ",
			"companyName": " TEST ",
		},
	}

	if err := validateCreateUserBusinessRules(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cmd.PasswordTemporary {
		t.Fatalf("expected password temporary to be forced true")
	}
	if cmd.IdentitySource != "local" {
		t.Fatalf("expected normalized identity source, got %q", cmd.IdentitySource)
	}
	if len(cmd.Groups) != 1 || cmd.Groups[0] != "/vpn/local" {
		t.Fatalf("expected normalized groups, got %#v", cmd.Groups)
	}
	if len(cmd.RequiredActions) != 1 || cmd.RequiredActions[0] != "UPDATE_PASSWORD" {
		t.Fatalf("expected normalized required actions, got %#v", cmd.RequiredActions)
	}
}

func TestValidateCreateUserBusinessRules_LDAPRequiresValidOnboardDateIfProvided(t *testing.T) {
	t.Parallel()

	cmd := &commands.CreateUser{
		Username:       "ldap.user",
		Email:          "ldap.user@example.com",
		FirstName:      "LDAP",
		LastName:       "User",
		IdentitySource: "ldap",
		Attributes: map[string]string{
			"fullName":    "LDAP User",
			"userType":    "employee",
			"phone":       "0987654321",
			"department":  "Phong Van hanh",
			"manager":     "CN=Manager,OU=Users,DC=mbfs,DC=local",
			"onboardDate": "2026-04-15",
		},
	}

	err := validateCreateUserBusinessRules(cmd)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	de, ok := err.(*domainerr.DomainError)
	if !ok {
		t.Fatalf("expected domain error, got %T", err)
	}
	if got := de.Details["attributes.onboardDate"]; got != "must be a valid date in dd/MM/yyyy" {
		t.Fatalf("expected onboard date detail, got %#v", de.Details)
	}
}

func TestValidateCreateUserBusinessRules_NormalizesOnboardAndWorkAddressAliases(t *testing.T) {
	t.Parallel()

	cmd := &commands.CreateUser{
		Username:       "ldap.user",
		Email:          "ldap.user@example.com",
		FirstName:      "LDAP",
		LastName:       "User",
		IdentitySource: "ldap",
		Attributes: map[string]string{
			"fullName":   " LDAP User ",
			"userType":   " employee ",
			"phone":      " 0987654321 ",
			"department": " Phong Van hanh ",
			"manager":    " CN=Manager,OU=Users,DC=mbfs,DC=local ",
			"onboard":    " 15/04/2026 ",
			"address":    " Toa nha MobiFone, Ha Noi ",
		},
	}

	if err := validateCreateUserBusinessRules(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cmd.Attributes["onboardDate"]; got != "15/04/2026" {
		t.Fatalf("expected normalized onboardDate, got %q", got)
	}
	if got := cmd.Attributes["workAddress"]; got != "Toa nha MobiFone, Ha Noi" {
		t.Fatalf("expected normalized workAddress, got %q", got)
	}
	if _, ok := cmd.Attributes["onboard"]; ok {
		t.Fatalf("expected onboard alias to be removed")
	}
	if _, ok := cmd.Attributes["address"]; ok {
		t.Fatalf("expected address alias to be removed")
	}
}

func TestUserServiceDeleteDeletesKeycloakUser(t *testing.T) {
	t.Parallel()

	called := false
	keycloak := &fakeKeycloakIdentityService{
		deleteUserFn: func(_ context.Context, id string) error {
			called = true
			if id != "user-id" {
				t.Fatalf("unexpected user id: %q", id)
			}
			return nil
		},
	}

	svc := NewUserService(validator.New(), keycloak, nil)
	if err := svc.Delete(context.Background(), "user-id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected keycloak delete to be called")
	}
}

func TestUserServiceCreateSendsAccountCreatedNotification(t *testing.T) {
	t.Parallel()

	var createdReq services.KeycloakUser
	keycloak := &fakeKeycloakIdentityService{
		searchUsersFn: func(context.Context, string, int, int) ([]services.KeycloakUser, error) {
			return nil, nil
		},
		createUserFn: func(_ context.Context, req services.KeycloakUser) (services.KeycloakUser, error) {
			createdReq = req
			return services.KeycloakUser{
				ID:             "user-1",
				Username:       req.Username,
				Email:          req.Email,
				DisplayName:    req.DisplayName,
				IdentitySource: req.IdentitySource,
				Enabled:        req.Enabled,
				Attributes:     cloneStringAttributes(req.Attributes),
			}, nil
		},
	}
	mailer := &fakeMailTemplateSender{enabled: true}
	notifications := NewNotificationService(mailer, nil, zerolog.Nop(), "Hệ thống SSO MBFS", "it.support@mbfs.vn", "https://sso.mbfs.vn")

	svc := NewUserService(validator.New(), keycloak, notifications)
	_, err := svc.Create(context.Background(), commands.CreateUser{
		Username:          "alice",
		Email:             "alice@example.com",
		NotificationEmail: "notify@example.com",
		FirstName:         "Alice",
		LastName:          "User",
		DisplayName:       "Alice User",
		IdentitySource:    "local",
		Enabled:           true,
		Password:          "Mbfs@111",
		Attributes: map[string]string{
			"fullName":    "Alice User",
			"userType":    "partner",
			"companyName": "TEST",
			"onboardDate": "15/04/2026",
			"workAddress": "Tòa nhà MobiFone, Hà Nội",
		},
		Actor: "api-admin",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mailer.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mailer.calls))
	}
	if got := mailer.calls[0].Template; got != "account_created" {
		t.Fatalf("unexpected template: %q", got)
	}
	if got := mailer.calls[0].To[0]; got != "notify@example.com" {
		t.Fatalf("unexpected recipient: %q", got)
	}
	data, ok := mailer.calls[0].Data.(accountCreatedTemplateData)
	if !ok {
		t.Fatalf("unexpected data type: %T", mailer.calls[0].Data)
	}
	if got := data.Email; got != "alice@example.com" {
		t.Fatalf("unexpected account email in mail data: %q", got)
	}
	if got := data.OnboardDate; got != "15/04/2026" {
		t.Fatalf("unexpected onboard date in mail data: %q", got)
	}
	if got := data.WorkAddress; got != "Tòa nhà MobiFone, Hà Nội" {
		t.Fatalf("unexpected work address in mail data: %q", got)
	}
	if _, ok := createdReq.Attributes["onboardDate"]; ok {
		t.Fatalf("expected onboarding-only attribute to be excluded from keycloak create payload")
	}
	if _, ok := createdReq.Attributes["workAddress"]; ok {
		t.Fatalf("expected work address to be excluded from keycloak create payload")
	}
}

func TestUserServiceCreateReturnsWarningWhenAccountEmailFails(t *testing.T) {
	t.Parallel()

	keycloak := &fakeKeycloakIdentityService{
		searchUsersFn: func(context.Context, string, int, int) ([]services.KeycloakUser, error) {
			return nil, nil
		},
		createUserFn: func(_ context.Context, req services.KeycloakUser) (services.KeycloakUser, error) {
			return services.KeycloakUser{
				ID:             "user-1",
				Username:       req.Username,
				Email:          req.Email,
				DisplayName:    req.DisplayName,
				IdentitySource: req.IdentitySource,
				Enabled:        req.Enabled,
				Attributes:     cloneStringAttributes(req.Attributes),
			}, nil
		},
	}
	mailer := &fakeMailTemplateSender{
		enabled: true,
		sendFn: func(context.Context, string, []string, any) error {
			return errors.New("smtp auth failed")
		},
	}
	notifications := NewNotificationService(mailer, nil, zerolog.Nop(), "Hệ thống SSO MBFS", "it.support@mbfs.vn", "https://sso.mbfs.vn")

	svc := NewUserService(validator.New(), keycloak, notifications)
	result, err := svc.Create(context.Background(), commands.CreateUser{
		Username:       "alice",
		Email:          "alice@example.com",
		FirstName:      "Alice",
		LastName:       "User",
		DisplayName:    "Alice User",
		IdentitySource: "local",
		Enabled:        true,
		Password:       "Mbfs@111",
		Attributes: map[string]string{
			"fullName":    "Alice User",
			"userType":    "partner",
			"companyName": "TEST",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %#v", result.Warnings)
	}
	if !strings.Contains(result.Warnings[0], "không gửi được email thông tin tài khoản") {
		t.Fatalf("unexpected warning: %q", result.Warnings[0])
	}
}

func TestUserServiceCreateGeneratesPasswordWhenMissing(t *testing.T) {
	t.Parallel()

	var createdReq services.KeycloakUser
	keycloak := &fakeKeycloakIdentityService{
		searchUsersFn: func(context.Context, string, int, int) ([]services.KeycloakUser, error) {
			return nil, nil
		},
		createUserFn: func(_ context.Context, req services.KeycloakUser) (services.KeycloakUser, error) {
			createdReq = req
			return services.KeycloakUser{
				ID:             "user-1",
				Username:       req.Username,
				Email:          req.Email,
				DisplayName:    req.DisplayName,
				IdentitySource: req.IdentitySource,
				Enabled:        req.Enabled,
				Attributes:     cloneStringAttributes(req.Attributes),
			}, nil
		},
	}
	mailer := &fakeMailTemplateSender{enabled: true}
	notifications := NewNotificationService(mailer, nil, zerolog.Nop(), "Hệ thống SSO MBFS", "it.support@mobifonesolutions.vn", "https://sso.mbfs.vn")

	svc := NewUserService(validator.New(), keycloak, notifications)
	_, err := svc.Create(context.Background(), commands.CreateUser{
		Username:       "alice",
		Email:          "alice@example.com",
		FirstName:      "Alice",
		LastName:       "User",
		DisplayName:    "Alice User",
		IdentitySource: "local",
		Enabled:        true,
		Attributes: map[string]string{
			"fullName":    "Alice User",
			"userType":    "partner",
			"companyName": "TEST",
		},
		Actor: "api-admin",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mailer.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mailer.calls))
	}
	if strings.TrimSpace(createdReq.Password) == "" {
		t.Fatal("expected generated password to be sent to keycloak")
	}
	if len(createdReq.Password) != 14 {
		t.Fatalf("expected generated password length 14, got %d", len(createdReq.Password))
	}
	if !strings.ContainsAny(createdReq.Password, "!@#$%*-_") {
		t.Fatalf("expected generated password to contain a symbol, got %q", createdReq.Password)
	}
	data, ok := mailer.calls[0].Data.(accountCreatedTemplateData)
	if !ok {
		t.Fatalf("unexpected data type: %T", mailer.calls[0].Data)
	}
	if data.TemporaryPassword != createdReq.Password {
		t.Fatalf("expected notification password to match generated password")
	}
}

func TestUserServiceCreateMapsKeycloak400ToInvalidArgument(t *testing.T) {
	t.Parallel()

	keycloak := &fakeKeycloakIdentityService{
		searchUsersFn: func(context.Context, string, int, int) ([]services.KeycloakUser, error) {
			return nil, nil
		},
		createUserFn: func(_ context.Context, req services.KeycloakUser) (services.KeycloakUser, error) {
			return services.KeycloakUser{}, errors.New(`create user failed: status=400 body={"errorMessage":"Unknown attribute onboardDate"}`)
		},
	}

	svc := NewUserService(validator.New(), keycloak, nil)
	_, err := svc.Create(context.Background(), commands.CreateUser{
		Username:       "alice",
		Email:          "alice@example.com",
		FirstName:      "Alice",
		LastName:       "User",
		DisplayName:    "Alice User",
		IdentitySource: "local",
		Enabled:        true,
		Password:       "Mbfs@111",
		Attributes: map[string]string{
			"fullName":    "Alice User",
			"userType":    "partner",
			"companyName": "TEST",
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	de, ok := err.(*domainerr.DomainError)
	if !ok {
		t.Fatalf("expected domain error, got %T", err)
	}
	if de.Code != domainerr.CodeInvalidArgument {
		t.Fatalf("expected invalid_argument, got %s", de.Code)
	}
	if de.Message != "keycloak rejected create user payload" {
		t.Fatalf("unexpected message: %q", de.Message)
	}
	if got := de.Details["upstream"]; got != "Unknown attribute onboardDate" {
		t.Fatalf("unexpected details: %#v", de.Details)
	}
}

func TestUserServiceUpdateEnabledSendsStatusNotification(t *testing.T) {
	t.Parallel()

	keycloak := &fakeKeycloakIdentityService{
		getUserFn: func(context.Context, string) (services.KeycloakUser, error) {
			return services.KeycloakUser{
				ID:          "user-1",
				Username:    "alice",
				Email:       "alice@example.com",
				DisplayName: "Alice",
				Enabled:     true,
			}, nil
		},
		updateUserFn: func(_ context.Context, id string, req services.KeycloakUser) (services.KeycloakUser, error) {
			return services.KeycloakUser{
				ID:          id,
				Username:    req.Username,
				Email:       req.Email,
				DisplayName: "Alice",
				Enabled:     req.Enabled,
			}, nil
		},
	}
	mailer := &fakeMailTemplateSender{enabled: true}
	notifications := NewNotificationService(mailer, nil, zerolog.Nop(), "Hệ thống SSO MBFS", "it.support@mbfs.vn", "https://sso.mbfs.vn")

	svc := NewUserService(validator.New(), keycloak, notifications)
	_, err := svc.Update(context.Background(), "user-1", commands.UpdateUser{
		Enabled: func() *bool { v := false; return &v }(),
		Actor:   "api-admin",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mailer.calls) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mailer.calls))
	}
	if got := mailer.calls[0].Template; got != "user_status_changed" {
		t.Fatalf("unexpected template: %q", got)
	}
}

func TestUserServiceUpdateSupportsKeycloakStyleUserRepresentationFields(t *testing.T) {
	t.Parallel()

	var captured services.KeycloakUser
	keycloak := &fakeKeycloakIdentityService{
		getUserFn: func(context.Context, string) (services.KeycloakUser, error) {
			return services.KeycloakUser{
				ID:              "user-1",
				Username:        "alice",
				Email:           "alice@example.com",
				DisplayName:     "User Alice",
				FirstName:       "Alice",
				LastName:        "User",
				Enabled:         true,
				EmailVerified:   false,
				RequiredActions: []string{"VERIFY_EMAIL"},
				Attributes: map[string]string{
					"fullName": "User Alice",
				},
			}, nil
		},
		updateUserFn: func(_ context.Context, _ string, req services.KeycloakUser) (services.KeycloakUser, error) {
			captured = req
			return req, nil
		},
	}

	svc := NewUserService(validator.New(), keycloak, nil)
	_, err := svc.Update(context.Background(), "user-1", commands.UpdateUser{
		Username:      func() *string { v := "alice.new"; return &v }(),
		Email:         func() *string { v := "alice.new@example.com"; return &v }(),
		FirstName:     func() *string { v := " Alice "; return &v }(),
		LastName:      func() *string { v := " Nguyen "; return &v }(),
		Enabled:       func() *bool { v := false; return &v }(),
		EmailVerified: func() *bool { v := true; return &v }(),
		RequiredActions: &[]string{
			" UPDATE_PASSWORD ",
			"",
		},
		Attributes: &map[string]string{
			"onboard":      "15/04/2026",
			"work_address": "Tòa nhà MobiFone, Hà Nội",
			"fullName":     " Alice Nguyen ",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := captured.Username; got != "alice.new" {
		t.Fatalf("unexpected username: %q", got)
	}
	if got := captured.Email; got != "alice.new@example.com" {
		t.Fatalf("unexpected email: %q", got)
	}
	if got := captured.FirstName; got != "Alice" {
		t.Fatalf("unexpected firstName: %q", got)
	}
	if got := captured.LastName; got != "Nguyen" {
		t.Fatalf("unexpected lastName: %q", got)
	}
	if captured.Enabled {
		t.Fatalf("expected enabled=false")
	}
	if !captured.EmailVerified {
		t.Fatalf("expected emailVerified=true")
	}
	if len(captured.RequiredActions) != 1 || captured.RequiredActions[0] != "UPDATE_PASSWORD" {
		t.Fatalf("unexpected required actions: %#v", captured.RequiredActions)
	}
	if got := captured.Attributes["onboardDate"]; got != "15/04/2026" {
		t.Fatalf("unexpected onboardDate: %q", got)
	}
	if got := captured.Attributes["workAddress"]; got != "Tòa nhà MobiFone, Hà Nội" {
		t.Fatalf("unexpected workAddress: %q", got)
	}
	if got := captured.Attributes["fullName"]; got != "Alice Nguyen" {
		t.Fatalf("unexpected fullName: %q", got)
	}
}

type fakeKeycloakIdentityService struct {
	createUserFn          func(context.Context, services.KeycloakUser) (services.KeycloakUser, error)
	updateUserFn          func(context.Context, string, services.KeycloakUser) (services.KeycloakUser, error)
	deleteUserFn          func(context.Context, string) error
	getUserFn             func(context.Context, string) (services.KeycloakUser, error)
	searchUsersFn         func(context.Context, string, int, int) ([]services.KeycloakUser, error)
	createGroupFn         func(context.Context, services.KeycloakGroup) (services.KeycloakGroup, error)
	updateGroupFn         func(context.Context, string, services.KeycloakGroup) (services.KeycloakGroup, error)
	deleteGroupFn         func(context.Context, string) error
	getGroupFn            func(context.Context, string) (services.KeycloakGroup, error)
	searchGroupsFn        func(context.Context, string) ([]services.KeycloakGroup, error)
	listGroupMembersFn    func(context.Context, string) ([]services.KeycloakUser, error)
	addUserToGroupFn      func(context.Context, string, string) error
	removeUserFromGroupFn func(context.Context, string, string) error
}

func (f *fakeKeycloakIdentityService) CreateUser(ctx context.Context, req services.KeycloakUser) (services.KeycloakUser, error) {
	if f.createUserFn != nil {
		return f.createUserFn(ctx, req)
	}
	return services.KeycloakUser{}, nil
}

func (f *fakeKeycloakIdentityService) UpdateUser(ctx context.Context, id string, req services.KeycloakUser) (services.KeycloakUser, error) {
	if f.updateUserFn != nil {
		return f.updateUserFn(ctx, id, req)
	}
	return services.KeycloakUser{}, nil
}

func (f *fakeKeycloakIdentityService) DeleteUser(ctx context.Context, id string) error {
	if f.deleteUserFn != nil {
		return f.deleteUserFn(ctx, id)
	}
	return nil
}

func (f *fakeKeycloakIdentityService) GetUser(ctx context.Context, id string) (services.KeycloakUser, error) {
	if f.getUserFn != nil {
		return f.getUserFn(ctx, id)
	}
	return services.KeycloakUser{}, nil
}

func (f *fakeKeycloakIdentityService) SearchUsers(ctx context.Context, q string, first, max int) ([]services.KeycloakUser, error) {
	if f.searchUsersFn != nil {
		return f.searchUsersFn(ctx, q, first, max)
	}
	return nil, nil
}

func (f *fakeKeycloakIdentityService) CreateGroup(ctx context.Context, req services.KeycloakGroup) (services.KeycloakGroup, error) {
	if f.createGroupFn != nil {
		return f.createGroupFn(ctx, req)
	}
	return services.KeycloakGroup{}, nil
}

func (f *fakeKeycloakIdentityService) UpdateGroup(ctx context.Context, id string, req services.KeycloakGroup) (services.KeycloakGroup, error) {
	if f.updateGroupFn != nil {
		return f.updateGroupFn(ctx, id, req)
	}
	return services.KeycloakGroup{}, nil
}

func (f *fakeKeycloakIdentityService) DeleteGroup(ctx context.Context, id string) error {
	if f.deleteGroupFn != nil {
		return f.deleteGroupFn(ctx, id)
	}
	return nil
}

func (f *fakeKeycloakIdentityService) GetGroup(ctx context.Context, id string) (services.KeycloakGroup, error) {
	if f.getGroupFn != nil {
		return f.getGroupFn(ctx, id)
	}
	return services.KeycloakGroup{}, nil
}

func (f *fakeKeycloakIdentityService) SearchGroups(ctx context.Context, q string) ([]services.KeycloakGroup, error) {
	if f.searchGroupsFn != nil {
		return f.searchGroupsFn(ctx, q)
	}
	return nil, nil
}

func (f *fakeKeycloakIdentityService) ListGroupMembers(ctx context.Context, groupID string) ([]services.KeycloakUser, error) {
	if f.listGroupMembersFn != nil {
		return f.listGroupMembersFn(ctx, groupID)
	}
	return nil, nil
}

func (f *fakeKeycloakIdentityService) AddUserToGroup(ctx context.Context, userID, groupID string) error {
	if f.addUserToGroupFn != nil {
		return f.addUserToGroupFn(ctx, userID, groupID)
	}
	return nil
}

func (f *fakeKeycloakIdentityService) RemoveUserFromGroup(ctx context.Context, userID, groupID string) error {
	if f.removeUserFromGroupFn != nil {
		return f.removeUserFromGroupFn(ctx, userID, groupID)
	}
	return nil
}

type fakeOpenVPNPolicyService struct {
	deleteUserFn       func(context.Context, string) error
	setUserGroupFn     func(context.Context, string, string) error
	ensureUserFn       func(context.Context, string) error
	ensureGroupFn      func(context.Context, string) error
	listUsersFn        func(context.Context, services.OpenVPNUserListQuery) (map[string]any, error)
	appendAccessListFn func(context.Context, []services.AccessRouteItem) error
}

func (f *fakeOpenVPNPolicyService) AddRuleset(context.Context, string, string) (int64, error) {
	return 0, nil
}

func (f *fakeOpenVPNPolicyService) UpdateRuleset(context.Context, int64, string, string) error {
	return nil
}

func (f *fakeOpenVPNPolicyService) DeleteRulesets(context.Context, []int64) error {
	return nil
}

func (f *fakeOpenVPNPolicyService) ListRulesets(context.Context, string, string) ([]services.OpenVPNRuleset, error) {
	return nil, nil
}

func (f *fakeOpenVPNPolicyService) ModifyRules(context.Context, []services.OpenVPNRule, []int64) ([]int64, error) {
	return nil, nil
}

func (f *fakeOpenVPNPolicyService) ModifyUserRulesetMapping(context.Context, map[string][]services.SubjectRulesetRef, map[string][]int64) error {
	return nil
}

func (f *fakeOpenVPNPolicyService) SetAccessList(context.Context, []services.AccessRouteItem) error {
	return nil
}

func (f *fakeOpenVPNPolicyService) AppendAccessList(ctx context.Context, items []services.AccessRouteItem) error {
	if f.appendAccessListFn != nil {
		return f.appendAccessListFn(ctx, items)
	}
	return nil
}

func (f *fakeOpenVPNPolicyService) RemoveAccessList(context.Context, []services.AccessRouteItem) error {
	return nil
}

func (f *fakeOpenVPNPolicyService) ListUsers(ctx context.Context, q services.OpenVPNUserListQuery) (map[string]any, error) {
	if f.listUsersFn != nil {
		return f.listUsersFn(ctx, q)
	}
	return nil, nil
}

func (f *fakeOpenVPNPolicyService) ListGroups(context.Context, services.OpenVPNGroupListQuery) (map[string]any, error) {
	return nil, nil
}

func (f *fakeOpenVPNPolicyService) ListAccessLists(context.Context, services.OpenVPNAccessListQuery) (map[string]any, error) {
	return nil, nil
}

func (f *fakeOpenVPNPolicyService) ListRules(context.Context, services.OpenVPNRuleListQuery) (map[string]any, error) {
	return nil, nil
}

func (f *fakeOpenVPNPolicyService) EnsureUser(ctx context.Context, username string) error {
	if f.ensureUserFn != nil {
		return f.ensureUserFn(ctx, username)
	}
	return nil
}

func (f *fakeOpenVPNPolicyService) EnsureGroup(ctx context.Context, groupname string) error {
	if f.ensureGroupFn != nil {
		return f.ensureGroupFn(ctx, groupname)
	}
	return nil
}

func (f *fakeOpenVPNPolicyService) SetUserGroup(ctx context.Context, username, groupname string) error {
	if f.setUserGroupFn != nil {
		return f.setUserGroupFn(ctx, username, groupname)
	}
	return nil
}

func (f *fakeOpenVPNPolicyService) DeleteUser(ctx context.Context, username string) error {
	if f.deleteUserFn != nil {
		return f.deleteUserFn(ctx, username)
	}
	return nil
}

func (f *fakeOpenVPNPolicyService) DeleteGroup(context.Context, string) error {
	return nil
}
