package application

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"

	"backend/internal/application/commands"
	domainerr "backend/internal/domain/errors"
	"backend/internal/domain/services"
)

type GroupService struct {
	validate *validator.Validate
	keycloak services.KeycloakIdentityService
}

func NewGroupService(validate *validator.Validate, keycloak services.KeycloakIdentityService) *GroupService {
	return &GroupService{
		validate: validate,
		keycloak: keycloak,
	}
}

func (s *GroupService) Create(ctx context.Context, cmd commands.CreateGroup) (services.KeycloakGroup, error) {
	if err := validatePayload(s.validate, cmd, "validation failed"); err != nil {
		return services.KeycloakGroup{}, err
	}
	group, err := s.keycloak.CreateGroup(ctx, services.KeycloakGroup{Name: cmd.Name})
	if err != nil {
		if isUpstreamStatus(err, 409) {
			return services.KeycloakGroup{}, domainerr.New(domainerr.CodeConflict, "group already exists on keycloak")
		}
		return services.KeycloakGroup{}, domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak create group failed", err)
	}
	return group, nil
}

func (s *GroupService) Update(ctx context.Context, id string, cmd commands.UpdateGroup) (services.KeycloakGroup, error) {
	if strings.TrimSpace(id) == "" {
		return services.KeycloakGroup{}, domainerr.New(domainerr.CodeInvalidArgument, "id is required")
	}
	if err := validatePayload(s.validate, cmd, "validation failed"); err != nil {
		return services.KeycloakGroup{}, err
	}

	current, err := s.Get(ctx, id)
	if err != nil {
		return services.KeycloakGroup{}, err
	}
	if cmd.Name != nil {
		current.Name = *cmd.Name
	}

	group, err := s.keycloak.UpdateGroup(ctx, id, current)
	if err != nil {
		if isUpstreamStatus(err, 404) {
			return services.KeycloakGroup{}, domainerr.New(domainerr.CodeNotFound, "group not found")
		}
		if isUpstreamStatus(err, 409) {
			return services.KeycloakGroup{}, domainerr.New(domainerr.CodeConflict, "group already exists on keycloak")
		}
		return services.KeycloakGroup{}, domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak update group failed", err)
	}
	return group, nil
}

func (s *GroupService) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return domainerr.New(domainerr.CodeInvalidArgument, "id is required")
	}
	if err := s.keycloak.DeleteGroup(ctx, id); err != nil {
		if isUpstreamStatus(err, 404) {
			return domainerr.New(domainerr.CodeNotFound, "group not found")
		}
		return domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak delete group failed", err)
	}
	return nil
}

func (s *GroupService) Get(ctx context.Context, id string) (services.KeycloakGroup, error) {
	if strings.TrimSpace(id) == "" {
		return services.KeycloakGroup{}, domainerr.New(domainerr.CodeInvalidArgument, "id is required")
	}
	group, err := s.keycloak.GetGroup(ctx, id)
	if err != nil {
		if isUpstreamStatus(err, 404) {
			return services.KeycloakGroup{}, domainerr.New(domainerr.CodeNotFound, "group not found")
		}
		return services.KeycloakGroup{}, domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak get group failed", err)
	}
	return group, nil
}

func (s *GroupService) List(ctx context.Context, q string) ([]services.KeycloakGroup, error) {
	groups, err := s.keycloak.SearchGroups(ctx, q)
	if err != nil {
		return nil, domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak search groups failed", err)
	}
	return groups, nil
}

func (s *GroupService) AddMember(ctx context.Context, groupID string, cmd commands.AddGroupMember) error {
	if strings.TrimSpace(groupID) == "" {
		return domainerr.New(domainerr.CodeInvalidArgument, "group id required")
	}
	if err := validatePayload(s.validate, cmd, "validation failed"); err != nil {
		return err
	}
	if err := s.keycloak.AddUserToGroup(ctx, cmd.UserID, groupID); err != nil {
		if isUpstreamStatus(err, 404) {
			return domainerr.New(domainerr.CodeNotFound, "user or group not found")
		}
		if isUpstreamStatus(err, 409) {
			return domainerr.New(domainerr.CodeConflict, "user is already a member of the group")
		}
		return domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak add member failed", err)
	}
	return nil
}

func (s *GroupService) RemoveMember(ctx context.Context, groupID, userID string) error {
	if strings.TrimSpace(groupID) == "" || strings.TrimSpace(userID) == "" {
		return domainerr.New(domainerr.CodeInvalidArgument, "group_id and user_id required")
	}
	if err := s.keycloak.RemoveUserFromGroup(ctx, userID, groupID); err != nil {
		if isUpstreamStatus(err, 404) {
			return domainerr.New(domainerr.CodeNotFound, "user or group membership not found")
		}
		return domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak remove member failed", err)
	}
	return nil
}
