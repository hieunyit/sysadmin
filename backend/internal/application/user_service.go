package application

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"

	"backend/internal/application/commands"
	domainerr "backend/internal/domain/errors"
	"backend/internal/domain/services"
	"backend/pkg/inputvalidate"
	"backend/pkg/passwordutil"
)

type UserService struct {
	validate      *validator.Validate
	keycloak      services.KeycloakIdentityService
	notifications *NotificationService
}

type CreateUserResult struct {
	User     services.KeycloakUser
	Warnings []string
}

func NewUserService(validate *validator.Validate, keycloak services.KeycloakIdentityService, notifications *NotificationService) *UserService {
	return &UserService{
		validate:      validate,
		keycloak:      keycloak,
		notifications: notifications,
	}
}

func (s *UserService) Create(ctx context.Context, cmd commands.CreateUser) (CreateUserResult, error) {
	if err := validatePayload(s.validate, cmd, "validation failed"); err != nil {
		return CreateUserResult{}, err
	}
	if err := validateCreateUserBusinessRules(&cmd); err != nil {
		return CreateUserResult{}, err
	}

	existing, err := s.findExistingKeycloakUser(ctx, cmd.Username, cmd.Email)
	if err != nil {
		return CreateUserResult{}, domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak pre-check user failed", err)
	}
	if existing != nil {
		return CreateUserResult{}, domainerr.New(domainerr.CodeConflict, "user already exists on keycloak")
	}

	displayName := strings.TrimSpace(cmd.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(strings.TrimSpace(cmd.LastName) + " " + strings.TrimSpace(cmd.FirstName))
	}

	attrs := make(map[string]string, len(cmd.Attributes)+1)
	for k, v := range cmd.Attributes {
		attrs[k] = v
	}
	if _, ok := attrs["fullName"]; !ok && displayName != "" {
		attrs["fullName"] = displayName
	}
	keycloakAttrs := cloneStringAttributes(attrs)
	delete(keycloakAttrs, "onboardDate")
	delete(keycloakAttrs, "workAddress")

	user, err := s.keycloak.CreateUser(ctx, services.KeycloakUser{
		Username:          cmd.Username,
		Email:             cmd.Email,
		DisplayName:       displayName,
		FirstName:         cmd.FirstName,
		LastName:          cmd.LastName,
		IdentitySource:    cmd.IdentitySource,
		Enabled:           cmd.Enabled,
		EmailVerified:     cmd.EmailVerified,
		RequiredActions:   cmd.RequiredActions,
		Attributes:        keycloakAttrs,
		Groups:            cmd.Groups,
		Password:          cmd.Password,
		PasswordTemporary: cmd.PasswordTemporary,
	})
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "unsupported identity_source="):
			return CreateUserResult{}, domainerr.New(domainerr.CodeInvalidArgument, err.Error())
		case isUpstreamStatus(err, 400):
			details := map[string]string{}
			if summary := upstreamSummary(err); summary != "" {
				details["upstream"] = summary
			}
			return CreateUserResult{}, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "keycloak rejected create user payload", details)
		case isUpstreamStatus(err, 409):
			return CreateUserResult{}, domainerr.New(domainerr.CodeConflict, "user already exists on keycloak")
		case strings.Contains(err.Error(), "requires KEYCLOAK_LDAP_COMPONENT_ID"),
			strings.Contains(err.Error(), "requires LDAP provider editMode to allow writes"),
			strings.Contains(err.Error(), "failed to restore LDAP component syncRegistrations"):
			return CreateUserResult{}, domainerr.New(domainerr.CodePreconditionFail, err.Error())
		}
		details := map[string]string{}
		if summary := upstreamSummary(err); summary != "" {
			details["upstream"] = summary
		}
		if len(details) > 0 {
			return CreateUserResult{}, domainerr.WrapWithDetails(domainerr.CodeExternalFailure, "keycloak create user failed", err, details)
		}
		return CreateUserResult{}, domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak create user failed", err)
	}

	var warnings []string
	if s.notifications != nil {
		user.IdentitySource = cmd.IdentitySource
		if user.Attributes == nil {
			user.Attributes = make(map[string]string)
		}
		for _, field := range []string{"onboardDate", "workAddress"} {
			if value := strings.TrimSpace(attrs[field]); value != "" {
				user.Attributes[field] = value
			}
		}
		warnings = appendWarnings(warnings, s.notifications.TrySendAccountCreated(ctx, user, cmd.Password, cmd.NotificationEmail)...)
	}
	return CreateUserResult{User: user, Warnings: warnings}, nil
}

func (s *UserService) Update(ctx context.Context, id string, cmd commands.UpdateUser) (services.KeycloakUser, error) {
	if strings.TrimSpace(id) == "" {
		return services.KeycloakUser{}, domainerr.New(domainerr.CodeInvalidArgument, "id is required")
	}
	if err := validatePayload(s.validate, cmd, "validation failed"); err != nil {
		return services.KeycloakUser{}, err
	}

	current, err := s.Get(ctx, id)
	if err != nil {
		return services.KeycloakUser{}, err
	}

	req := services.KeycloakUser{
		Username:        current.Username,
		Email:           current.Email,
		DisplayName:     current.DisplayName,
		Enabled:         current.Enabled,
		EmailVerified:   current.EmailVerified,
		FirstName:       current.FirstName,
		LastName:        current.LastName,
		RequiredActions: append([]string(nil), current.RequiredActions...),
		Attributes:      cloneStringAttributes(current.Attributes),
	}
	if cmd.Username != nil {
		req.Username = strings.TrimSpace(*cmd.Username)
		if req.Username == "" {
			return services.KeycloakUser{}, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"username": "must not be blank",
			})
		}
	}
	if cmd.Email != nil {
		req.Email = strings.TrimSpace(*cmd.Email)
		if req.Email == "" {
			return services.KeycloakUser{}, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"email": "must not be blank",
			})
		}
	}
	if cmd.DisplayName != nil {
		req.DisplayName = strings.TrimSpace(*cmd.DisplayName)
		if req.DisplayName == "" {
			return services.KeycloakUser{}, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"display_name": "must not be blank",
			})
		}
		req.FirstName, req.LastName = splitDisplayName(req.DisplayName)
	}
	if cmd.FirstName != nil {
		req.FirstName = strings.TrimSpace(*cmd.FirstName)
		if req.FirstName == "" {
			return services.KeycloakUser{}, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"first_name": "must not be blank",
			})
		}
	}
	if cmd.LastName != nil {
		req.LastName = strings.TrimSpace(*cmd.LastName)
		if req.LastName == "" {
			return services.KeycloakUser{}, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"last_name": "must not be blank",
			})
		}
	}
	if cmd.FirstName != nil || cmd.LastName != nil {
		req.DisplayName = strings.TrimSpace(strings.TrimSpace(req.LastName) + " " + strings.TrimSpace(req.FirstName))
	}
	if cmd.Enabled != nil {
		req.Enabled = *cmd.Enabled
	}
	if cmd.EmailVerified != nil {
		req.EmailVerified = *cmd.EmailVerified
	}
	if cmd.RequiredActions != nil {
		req.RequiredActions = normalizeStringSlice(*cmd.RequiredActions)
	}
	if cmd.Attributes != nil {
		attrs := cloneStringAttributes(*cmd.Attributes)
		if attrs == nil {
			attrs = map[string]string{}
		}
		for key, value := range attrs {
			trimmedKey := strings.TrimSpace(key)
			trimmedValue := strings.TrimSpace(value)
			if trimmedKey == "" || trimmedValue == "" {
				delete(attrs, key)
				continue
			}
			if trimmedKey != key {
				delete(attrs, key)
				attrs[trimmedKey] = trimmedValue
			} else {
				attrs[key] = trimmedValue
			}
		}
		normalizeAttributeAlias(attrs, "onboardDate", "onboard", "onboard_date")
		normalizeAttributeAlias(attrs, "workAddress", "address", "work_address")
		if onboardDate := strings.TrimSpace(attrs["onboardDate"]); onboardDate != "" {
			normalizedOnboard, err := inputvalidate.NormalizeDDMMYYYY(onboardDate)
			if err != nil {
				return services.KeycloakUser{}, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
					"attributes.onboardDate": "must be a valid date in dd/MM/yyyy",
				})
			}
			attrs["onboardDate"] = normalizedOnboard
		}
		req.Attributes = attrs
	}

	updated, err := s.keycloak.UpdateUser(ctx, id, req)
	if err != nil {
		if isUpstreamStatus(err, 404) {
			return services.KeycloakUser{}, domainerr.New(domainerr.CodeNotFound, "user not found")
		}
		if isUpstreamStatus(err, 400) {
			details := map[string]string{}
			if summary := upstreamSummary(err); summary != "" {
				details["upstream"] = summary
			}
			return services.KeycloakUser{}, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "keycloak rejected update user payload", details)
		}
		return services.KeycloakUser{}, domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak update user failed", err)
	}
	if s.notifications != nil && cmd.Enabled != nil && current.Enabled != updated.Enabled {
		s.notifications.TrySendUserStatusChanged(ctx, updated, cmd.Actor)
	}
	return updated, nil
}

func (s *UserService) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return domainerr.New(domainerr.CodeInvalidArgument, "id is required")
	}
	if err := s.keycloak.DeleteUser(ctx, id); err != nil {
		if isUpstreamStatus(err, 404) {
			return domainerr.New(domainerr.CodeNotFound, "user not found")
		}
		return domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak delete user failed", err)
	}
	return nil
}

func appendWarnings(current []string, warnings ...string) []string {
	seen := make(map[string]struct{}, len(current))
	for _, warning := range current {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}
		seen[warning] = struct{}{}
	}
	for _, warning := range warnings {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}
		if _, ok := seen[warning]; ok {
			continue
		}
		seen[warning] = struct{}{}
		current = append(current, warning)
	}
	return current
}

func (s *UserService) Get(ctx context.Context, id string) (services.KeycloakUser, error) {
	if strings.TrimSpace(id) == "" {
		return services.KeycloakUser{}, domainerr.New(domainerr.CodeInvalidArgument, "id is required")
	}
	user, err := s.keycloak.GetUser(ctx, id)
	if err != nil {
		if isUpstreamStatus(err, 404) {
			return services.KeycloakUser{}, domainerr.New(domainerr.CodeNotFound, "user not found")
		}
		return services.KeycloakUser{}, domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak get user failed", err)
	}
	return user, nil
}

func (s *UserService) List(ctx context.Context, q string, first, max int) ([]services.KeycloakUser, error) {
	if max <= 0 {
		max = 50
	}
	if max > 500 {
		max = 500
	}
	users, err := s.keycloak.SearchUsers(ctx, q, first, max)
	if err != nil {
		return nil, domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak search users failed", err)
	}
	return users, nil
}

func validateCreateUserBusinessRules(cmd *commands.CreateUser) error {
	if cmd == nil {
		return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
			"body": "create user payload is required",
		})
	}

	if err := normalizeCreateUserInput(cmd); err != nil {
		return err
	}

	identitySource := strings.ToLower(strings.TrimSpace(cmd.IdentitySource))
	attrs := cmd.Attributes

	if attrs != nil && strings.TrimSpace(attrs["onboardDate"]) != "" {
		normalizedOnboard, err := inputvalidate.NormalizeDDMMYYYY(attrs["onboardDate"])
		if err != nil {
			return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"attributes.onboardDate": "must be a valid date in dd/MM/yyyy",
			})
		}
		attrs["onboardDate"] = normalizedOnboard
	}

	fullName := ""
	if attrs != nil {
		fullName = strings.TrimSpace(attrs["fullName"])
	}
	if fullName == "" {
		return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
			"attributes.fullName": "is required",
		})
	}

	userType := ""
	if attrs != nil {
		userType = strings.ToLower(strings.TrimSpace(attrs["userType"]))
	}

	switch identitySource {
	case "ldap":
		cmd.PasswordTemporary = true
		if userType != "employee" {
			return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"attributes.userType": "must be employee for ldap users",
			})
		}
		for _, field := range []string{"phone", "department", "manager"} {
			if attrs == nil || strings.TrimSpace(attrs[field]) == "" {
				return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
					"attributes." + field: "is required for ldap users",
				})
			}
		}
	case "local":
		cmd.PasswordTemporary = true
		if userType != "partner" && userType != "outsource" {
			return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"attributes.userType": "must be partner or outsource for local users",
			})
		}
		for _, field := range []string{"companyName"} {
			if attrs == nil || strings.TrimSpace(attrs[field]) == "" {
				return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
					"attributes." + field: "is required for local users",
				})
			}
		}
	}

	return nil
}

func normalizeCreateUserInput(cmd *commands.CreateUser) error {
	cmd.Username = strings.TrimSpace(cmd.Username)
	cmd.Email = strings.TrimSpace(cmd.Email)
	cmd.NotificationEmail = strings.TrimSpace(cmd.NotificationEmail)
	cmd.FirstName = strings.TrimSpace(cmd.FirstName)
	cmd.LastName = strings.TrimSpace(cmd.LastName)
	cmd.DisplayName = strings.TrimSpace(cmd.DisplayName)
	cmd.IdentitySource = strings.ToLower(strings.TrimSpace(cmd.IdentitySource))
	cmd.Password = strings.TrimSpace(cmd.Password)

	if cmd.Attributes != nil {
		for key, value := range cmd.Attributes {
			cmd.Attributes[key] = strings.TrimSpace(value)
		}
		normalizeAttributeAlias(cmd.Attributes, "onboardDate", "onboard", "onboard_date")
		normalizeAttributeAlias(cmd.Attributes, "workAddress", "address", "work_address")
	}

	if len(cmd.Groups) > 0 {
		groups := make([]string, 0, len(cmd.Groups))
		for _, group := range cmd.Groups {
			group = strings.TrimSpace(group)
			if group == "" {
				continue
			}
			groups = append(groups, group)
		}
		cmd.Groups = groups
	}

	if len(cmd.RequiredActions) > 0 {
		actions := make([]string, 0, len(cmd.RequiredActions))
		for _, action := range cmd.RequiredActions {
			action = strings.TrimSpace(action)
			if action == "" {
				continue
			}
			actions = append(actions, action)
		}
		cmd.RequiredActions = actions
	}

	if cmd.Password == "" {
		generatedPassword, err := passwordutil.GenerateTemporary()
		if err != nil {
			return domainerr.Wrap(domainerr.CodeInternal, "generate temporary password failed", err)
		}
		cmd.Password = generatedPassword
	}

	return nil
}

func splitDisplayName(display string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(display))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[len(parts)-1], strings.Join(parts[:len(parts)-1], " ")
}

func normalizeStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func normalizeAttributeAlias(attrs map[string]string, canonical string, aliases ...string) {
	if attrs == nil {
		return
	}
	if strings.TrimSpace(attrs[canonical]) != "" {
		for _, alias := range aliases {
			delete(attrs, alias)
		}
		return
	}
	for _, alias := range aliases {
		if value := strings.TrimSpace(attrs[alias]); value != "" {
			attrs[canonical] = value
			break
		}
	}
	for _, alias := range aliases {
		delete(attrs, alias)
	}
}

func cloneStringAttributes(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (s *UserService) findExistingKeycloakUser(ctx context.Context, username, email string) (*services.KeycloakUser, error) {
	queries := []string{strings.TrimSpace(username)}
	if e := strings.TrimSpace(email); e != "" && !strings.EqualFold(e, username) {
		queries = append(queries, e)
	}

	seen := map[string]struct{}{}
	for _, q := range queries {
		if q == "" {
			continue
		}
		first := 0
		const pageSize = 100
		for {
			users, err := s.keycloak.SearchUsers(ctx, q, first, pageSize)
			if err != nil {
				return nil, err
			}
			for _, u := range users {
				if strings.TrimSpace(u.ID) == "" {
					continue
				}
				if _, ok := seen[u.ID]; ok {
					continue
				}
				seen[u.ID] = struct{}{}

				if strings.EqualFold(strings.TrimSpace(u.Username), strings.TrimSpace(username)) {
					user := u
					return &user, nil
				}
				if strings.TrimSpace(email) != "" && strings.EqualFold(strings.TrimSpace(u.Email), strings.TrimSpace(email)) {
					user := u
					return &user, nil
				}
			}
			if len(users) < pageSize {
				break
			}
			first += pageSize
		}
	}
	return nil, nil
}
