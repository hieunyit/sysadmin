package application

import (
	"context"
	"fmt"
	"strings"

	domainerr "backend/internal/domain/errors"
	"backend/internal/domain/services"
)

type OpenVPNAdminService struct {
	ovpn          services.OpenVPNPolicyService
	notifications *NotificationService
}

func NewOpenVPNAdminService(ovpn services.OpenVPNPolicyService, notifications *NotificationService) *OpenVPNAdminService {
	return &OpenVPNAdminService{ovpn: ovpn, notifications: notifications}
}

func (s *OpenVPNAdminService) ListUsers(ctx context.Context, q services.OpenVPNUserListQuery) (map[string]any, error) {
	out, err := s.ovpn.ListUsers(ctx, q)
	if err != nil {
		return nil, domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn users list failed", err)
	}
	return sanitizeOpenVPNUserProfiles(out)
}

func (s *OpenVPNAdminService) ListGroups(ctx context.Context, q services.OpenVPNGroupListQuery) (map[string]any, error) {
	out, err := s.ovpn.ListGroups(ctx, q)
	if err != nil {
		return nil, domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn groups list failed", err)
	}
	return out, nil
}

func (s *OpenVPNAdminService) ListAllUsers(ctx context.Context, q services.OpenVPNUserListQuery) (map[string]any, error) {
	out, err := s.listAllUsers(ctx, q)
	if err != nil {
		return nil, err
	}
	return sanitizeOpenVPNUserProfiles(out)
}

func (s *OpenVPNAdminService) ListAllGroups(ctx context.Context, q services.OpenVPNGroupListQuery) (map[string]any, error) {
	return s.listAllGroups(ctx, q)
}

func (s *OpenVPNAdminService) ExportUsers(ctx context.Context, q string) ([]services.OpenVPNUserExportRow, []string, error) {
	out, err := s.listAllUsers(ctx, services.OpenVPNUserListQuery{
		Limit:  500,
		Offset: 0,
		Search: q,
	})
	if err != nil {
		return nil, nil, err
	}

	profiles, err := extractProfiles(out)
	if err != nil {
		return nil, nil, domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn users export failed", err)
	}

	rows := make([]services.OpenVPNUserExportRow, 0, len(profiles))
	for _, profile := range profiles {
		username := strings.TrimSpace(formatMapString(profile, "name"))
		if username == "" {
			continue
		}
		rows = append(rows, services.OpenVPNUserExportRow{
			Username:        username,
			OpenVPNGroup:    strings.TrimSpace(formatMapString(profile, "group")),
			AuthMethod:      propertyValueString(profile["auth_method"]),
			Deny:            propertyValueString(profile["deny"]),
			PasswordDefined: strings.TrimSpace(formatMapString(profile, "password_defined")),
			MFAStatus:       strings.TrimSpace(formatMapString(profile, "mfa_status")),
		})
	}
	return rows, nil, nil
}

func sanitizeOpenVPNUserProfiles(out map[string]any) (map[string]any, error) {
	profiles, err := extractProfiles(out)
	if err != nil {
		return nil, domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn users list failed", err)
	}
	if len(profiles) == 0 {
		return out, nil
	}
	enriched := make([]any, 0, len(profiles))
	for _, profile := range profiles {
		copyProfile := copyStringAnyMap(profile)
		delete(copyProfile, "admin")
		enriched = append(enriched, copyProfile)
	}

	result := copyStringAnyMap(out)
	result["profiles"] = enriched
	return result, nil
}

func (s *OpenVPNAdminService) listAllUsers(ctx context.Context, q services.OpenVPNUserListQuery) (map[string]any, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 500
	}

	allProfiles := make([]any, 0)
	offset := q.Offset
	for {
		page, err := s.ovpn.ListUsers(ctx, services.OpenVPNUserListQuery{
			Limit:  limit,
			Offset: offset,
			Search: q.Search,
		})
		if err != nil {
			return nil, domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn users list failed", err)
		}

		profiles, err := extractProfiles(page)
		if err != nil {
			return nil, domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn users list failed", err)
		}
		if len(profiles) == 0 {
			break
		}
		for _, profile := range profiles {
			allProfiles = append(allProfiles, profile)
		}
		if len(profiles) < limit {
			break
		}
		offset += limit
	}

	return map[string]any{
		"profiles": allProfiles,
		"total":    len(allProfiles),
	}, nil
}

func (s *OpenVPNAdminService) listAllGroups(ctx context.Context, q services.OpenVPNGroupListQuery) (map[string]any, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 500
	}

	allProfiles := make([]any, 0)
	offset := q.Offset
	for {
		page, err := s.ovpn.ListGroups(ctx, services.OpenVPNGroupListQuery{
			Limit:            limit,
			Offset:           offset,
			Search:           q.Search,
			EnumerateMembers: q.EnumerateMembers,
		})
		if err != nil {
			return nil, domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn groups list failed", err)
		}

		profiles, err := extractProfiles(page)
		if err != nil {
			return nil, domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn groups list failed", err)
		}
		if len(profiles) == 0 {
			break
		}
		for _, profile := range profiles {
			allProfiles = append(allProfiles, profile)
		}
		if len(profiles) < limit {
			break
		}
		offset += limit
	}

	return map[string]any{
		"profiles": allProfiles,
		"total":    len(allProfiles),
	}, nil
}

func extractProfiles(out map[string]any) ([]map[string]any, error) {
	if out == nil {
		return nil, nil
	}
	raw, ok := out["profiles"]
	if !ok || raw == nil {
		return []map[string]any{}, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, domainerr.New(domainerr.CodeExternalFailure, "unexpected openvpn response: profiles is not an array")
	}
	profiles := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		profiles = append(profiles, row)
	}
	return profiles, nil
}

func formatMapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func propertyValueString(v any) string {
	if v == nil {
		return ""
	}
	if m, ok := v.(map[string]any); ok {
		if value, exists := m["value"]; exists {
			return strings.TrimSpace(fmt.Sprintf("%v", value))
		}
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func copyStringAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
