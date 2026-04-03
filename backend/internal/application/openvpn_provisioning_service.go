package application

import (
	"context"
	"strings"

	domainerr "backend/internal/domain/errors"
	"backend/internal/domain/services"
)

type OpenVPNProvisioningService struct {
	ovpn          services.OpenVPNPolicyService
	keycloak      services.KeycloakIdentityService
	notifications *NotificationService
}

type CreateOpenVPNUserFromKeycloakResult struct {
	User           services.KeycloakUser
	OpenVPNUser    string
	OpenVPNGroup   string
	Warnings       []string
	UsedKeycloakID string
}

func NewOpenVPNProvisioningService(ovpn services.OpenVPNPolicyService, keycloak services.KeycloakIdentityService, notifications *NotificationService) *OpenVPNProvisioningService {
	return &OpenVPNProvisioningService{
		ovpn:          ovpn,
		keycloak:      keycloak,
		notifications: notifications,
	}
}

func (s *OpenVPNProvisioningService) CreateUserFromKeycloak(ctx context.Context, userID, username, vpnGroup string) (CreateOpenVPNUserFromKeycloakResult, error) {
	if s.ovpn == nil {
		return CreateOpenVPNUserFromKeycloakResult{}, domainerr.New(domainerr.CodePreconditionFail, "openvpn integration is not configured")
	}
	if s.keycloak == nil {
		return CreateOpenVPNUserFromKeycloakResult{}, domainerr.New(domainerr.CodePreconditionFail, "keycloak integration is not configured")
	}

	userID = strings.TrimSpace(userID)
	username = strings.TrimSpace(username)
	vpnGroup = strings.TrimSpace(vpnGroup)
	if userID == "" && username == "" {
		return CreateOpenVPNUserFromKeycloakResult{}, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
			"user_id":  "either user_id or username is required",
			"username": "either username or user_id is required",
		})
	}

	kcUser, err := s.resolveKeycloakUser(ctx, userID, username)
	if err != nil {
		return CreateOpenVPNUserFromKeycloakResult{}, err
	}
	if strings.TrimSpace(kcUser.Username) == "" {
		return CreateOpenVPNUserFromKeycloakResult{}, domainerr.New(domainerr.CodePreconditionFail, "keycloak user has empty username")
	}
	if !kcUser.Enabled {
		return CreateOpenVPNUserFromKeycloakResult{}, domainerr.New(domainerr.CodePreconditionFail, "keycloak user is disabled")
	}

	if err := s.ovpn.EnsureUser(ctx, kcUser.Username); err != nil {
		return CreateOpenVPNUserFromKeycloakResult{}, domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn user create failed", err)
	}
	if vpnGroup != "" {
		if err := s.ovpn.EnsureGroup(ctx, vpnGroup); err != nil {
			return CreateOpenVPNUserFromKeycloakResult{}, domainerr.WrapWithDetails(domainerr.CodeExternalFailure, "openvpn group create failed", err, map[string]string{
				"vpn_group": vpnGroup,
			})
		}
		if err := s.ovpn.SetUserGroup(ctx, kcUser.Username, vpnGroup); err != nil {
			return CreateOpenVPNUserFromKeycloakResult{}, domainerr.WrapWithDetails(domainerr.CodeExternalFailure, "openvpn user group assignment failed", err, map[string]string{
				"username":  kcUser.Username,
				"vpn_group": vpnGroup,
			})
		}
	}

	var warnings []string
	if s.notifications != nil {
		warnings = appendWarnings(warnings, s.notifications.TrySendVPNProvisioned(ctx, kcUser, vpnGroup)...)
	}

	return CreateOpenVPNUserFromKeycloakResult{
		User:           kcUser,
		OpenVPNUser:    strings.TrimSpace(kcUser.Username),
		OpenVPNGroup:   vpnGroup,
		Warnings:       warnings,
		UsedKeycloakID: strings.TrimSpace(kcUser.ID),
	}, nil
}

func (s *OpenVPNProvisioningService) resolveKeycloakUser(ctx context.Context, userID, username string) (services.KeycloakUser, error) {
	if userID != "" {
		user, err := s.keycloak.GetUser(ctx, userID)
		if err != nil {
			if isUpstreamStatus(err, 404) {
				return services.KeycloakUser{}, domainerr.New(domainerr.CodeNotFound, "keycloak user not found")
			}
			return services.KeycloakUser{}, domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak get user failed", err)
		}
		if username != "" && !strings.EqualFold(strings.TrimSpace(user.Username), username) {
			return services.KeycloakUser{}, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"username": "does not match user_id",
			})
		}
		return user, nil
	}

	users, err := s.keycloak.SearchUsers(ctx, username, 0, 50)
	if err != nil {
		return services.KeycloakUser{}, domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak search users failed", err)
	}
	for _, user := range users {
		if strings.EqualFold(strings.TrimSpace(user.Username), username) {
			return user, nil
		}
	}
	return services.KeycloakUser{}, domainerr.New(domainerr.CodeNotFound, "keycloak user not found")
}
