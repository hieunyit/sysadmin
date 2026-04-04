package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"backend/internal/application"
	domainerr "backend/internal/domain/errors"
	"backend/pkg/httputil"
)

func (h *Handler) ensureAdminPortal() (*application.AdminPortalService, error) {
	if h == nil || h.services == nil || h.services.AdminPortal == nil {
		return nil, domainerr.New(domainerr.CodePreconditionFail, "admin portal backend is not configured")
	}
	return h.services.AdminPortal, nil
}

func bodyString(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func bodyBool(data map[string]any, key string) bool {
	if data == nil {
		return false
	}
	value, ok := data[key]
	if !ok || value == nil {
		return false
	}
	out, _ := value.(bool)
	return out
}

func bodyRoleRefs(data map[string]any, key string) []application.RoleRef {
	raw, ok := data[key].([]any)
	if !ok {
		return nil
	}
	out := make([]application.RoleRef, 0, len(raw))
	for _, item := range raw {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, application.RoleRef{
			ID:   bodyString(obj, "id"),
			Name: bodyString(obj, "name"),
		})
	}
	return out
}

func (h *Handler) GetAdminDashboard(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	data, err := service.GetDashboardSummary(r.Context())
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: data})
}

func (h *Handler) ListAdminAuditLogs(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	limit, offset, err := parseLimitOffset(r, 50)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	items, total, err := service.ListAuditLogs(r.Context(), application.AuditLogFilter{
		Limit:      limit,
		Offset:     offset,
		Search:     strings.TrimSpace(r.URL.Query().Get("search")),
		Source:     strings.TrimSpace(r.URL.Query().Get("source")),
		Status:     strings.TrimSpace(r.URL.Query().Get("status")),
		ActionType: strings.TrimSpace(r.URL.Query().Get("action_type")),
	})
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{
		Data: map[string]any{
			"logs":  items,
			"total": total,
		},
	})
}

func (h *Handler) AdminListKeycloakUsers(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	first := 0
	max := 20
	if v := strings.TrimSpace(r.URL.Query().Get("first")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			first = parsed
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("max")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			max = parsed
		}
	}
	users, total, err := service.ListKeycloakUsers(r.Context(), strings.TrimSpace(r.URL.Query().Get("search")), first, max)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"users": users, "total": total}})
}

func (h *Handler) AdminListKeycloakGroups(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	groups, err := service.ListKeycloakGroups(r.Context())
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"groups": groups}})
}

func (h *Handler) AdminListKeycloakRoles(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	roles, err := service.ListKeycloakRoles(r.Context())
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"roles": roles}})
}

func (h *Handler) AdminListKeycloakSessions(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	sessions, err := service.ListKeycloakSessions(r.Context())
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"sessions": sessions}})
}

func (h *Handler) AdminListOpenVPNUsers(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	users, err := service.ListOpenVPNUsers(r.Context())
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"users": users}})
}

func (h *Handler) AdminListOpenVPNGroups(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	groups, err := service.ListOpenVPNGroups(r.Context())
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"groups": groups}})
}

func (h *Handler) AdminListOpenVPNConnections(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	history := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("type")), "history")
	connections, err := service.ListOpenVPNConnections(r.Context(), history)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"connections": connections}})
}

func (h *Handler) AdminListOpenVPNConfigs(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	var username *string
	if value := strings.TrimSpace(r.URL.Query().Get("username")); value != "" {
		username = &value
	}
	configs, err := service.ListOpenVPNConfigs(r.Context(), username)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"configs": configs}})
}

func (h *Handler) AdminKeycloakUsersAction(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	var body map[string]any
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	action := bodyString(body, "action")
	switch action {
	case "create":
		user, err := service.CreateKeycloakUser(r.Context(), application.KeycloakUserCreateInput{
			Username:          bodyString(body, "username"),
			Email:             bodyString(body, "email"),
			FirstName:         bodyString(body, "firstName"),
			LastName:          bodyString(body, "lastName"),
			Password:          bodyString(body, "password"),
			Enabled:           bodyBool(body, "enabled"),
			TemporaryPassword: bodyBool(body, "temporaryPassword"),
		}, actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true, "id": user.ID, "user": user}})
	case "update":
		user, err := service.UpdateKeycloakUser(r.Context(), bodyString(body, "id"), application.KeycloakUserUpdateInput{
			Email:     optionalString(body, "email"),
			FirstName: optionalString(body, "firstName"),
			LastName:  optionalString(body, "lastName"),
			Enabled:   optionalBool(body, "enabled"),
		}, actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true, "user": user}})
	case "delete":
		err := service.DeleteKeycloakUser(r.Context(), bodyString(body, "id"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "resetPassword":
		err := service.ResetKeycloakUserPassword(r.Context(), bodyString(body, "id"), bodyString(body, "password"), bodyBool(body, "temporary"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "getRoles":
		roles, err := service.GetKeycloakUserRoles(r.Context(), bodyString(body, "id"))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"roles": roles}})
	case "assignRole":
		err := service.AssignRolesToKeycloakUser(r.Context(), bodyString(body, "userId"), bodyRoleRefs(body, "roles"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "removeRole":
		err := service.RemoveRolesFromKeycloakUser(r.Context(), bodyString(body, "userId"), bodyRoleRefs(body, "roles"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "getGroups":
		groups, err := service.GetKeycloakUserGroups(r.Context(), bodyString(body, "id"))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"groups": groups}})
	case "addToGroup":
		err := service.AddKeycloakUserToGroup(r.Context(), bodyString(body, "userId"), bodyString(body, "groupId"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "removeFromGroup":
		err := service.RemoveKeycloakUserFromGroup(r.Context(), bodyString(body, "userId"), bodyString(body, "groupId"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "logout":
		err := service.LogoutKeycloakUser(r.Context(), bodyString(body, "id"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "sendVerifyEmail":
		err := service.SendKeycloakVerifyEmail(r.Context(), bodyString(body, "id"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	default:
		httputil.WriteError(w, domainerr.New(domainerr.CodeInvalidArgument, "invalid action"), requestID(r))
	}
}

func (h *Handler) AdminKeycloakGroupsAction(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	var body map[string]any
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	action := bodyString(body, "action")
	switch action {
	case "create":
		group, err := service.CreateKeycloakGroup(r.Context(), bodyString(body, "name"), optionalString(body, "parentId"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true, "id": group.ID, "group": group}})
	case "update":
		group, err := service.UpdateKeycloakGroup(r.Context(), bodyString(body, "id"), bodyString(body, "name"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true, "group": group}})
	case "delete":
		err := service.DeleteKeycloakGroup(r.Context(), bodyString(body, "id"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "getMembers":
		members, err := service.GetKeycloakGroupMembers(r.Context(), bodyString(body, "id"))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"members": members}})
	case "getRoles":
		roles, err := service.GetKeycloakGroupRoles(r.Context(), bodyString(body, "id"))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"roles": roles}})
	case "assignRole":
		err := service.AssignRolesToKeycloakGroup(r.Context(), bodyString(body, "groupId"), bodyRoleRefs(body, "roles"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "removeRole":
		err := service.RemoveRolesFromKeycloakGroup(r.Context(), bodyString(body, "groupId"), bodyRoleRefs(body, "roles"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	default:
		httputil.WriteError(w, domainerr.New(domainerr.CodeInvalidArgument, "invalid action"), requestID(r))
	}
}

func (h *Handler) AdminKeycloakRolesAction(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	var body map[string]any
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	switch bodyString(body, "action") {
	case "create":
		role, err := service.CreateKeycloakRole(r.Context(), bodyString(body, "name"), bodyString(body, "description"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true, "role": role}})
	case "delete":
		err := service.DeleteKeycloakRole(r.Context(), bodyString(body, "name"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	default:
		httputil.WriteError(w, domainerr.New(domainerr.CodeInvalidArgument, "invalid action"), requestID(r))
	}
}

func (h *Handler) AdminKeycloakSessionsAction(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	var body map[string]any
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	switch bodyString(body, "action") {
	case "logoutSession":
		err := service.LogoutKeycloakSession(r.Context(), bodyString(body, "sessionId"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
	case "logoutUser":
		err := service.LogoutKeycloakUser(r.Context(), bodyString(body, "userId"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
	case "logoutAllSessions":
		err := service.LogoutAllKeycloakSessions(r.Context(), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
	default:
		httputil.WriteError(w, domainerr.New(domainerr.CodeInvalidArgument, "invalid action"), requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
}

func (h *Handler) AdminOpenVPNUsersAction(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	var body map[string]any
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	switch bodyString(body, "action") {
	case "create":
		user, err := service.CreateOpenVPNUser(r.Context(), application.OpenVPNUserCreateInput{
			Username:      bodyString(body, "username"),
			Email:         bodyString(body, "email"),
			Group:         optionalString(body, "group"),
			Enabled:       bodyBool(body, "enabled"),
			PropAutologin: bodyBool(body, "prop_autologin"),
			PropAdmin:     bodyBool(body, "prop_admin"),
		}, actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true, "user": user}})
	case "update":
		user, err := service.UpdateOpenVPNUser(r.Context(), bodyString(body, "username"), application.OpenVPNUserUpdateInput{
			Email:   optionalString(body, "email"),
			Enabled: optionalBool(body, "enabled"),
		}, actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true, "user": user}})
	case "delete":
		err := service.DeleteOpenVPNUser(r.Context(), bodyString(body, "username"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "enable":
		if err := service.EnableOpenVPNUser(r.Context(), bodyString(body, "username"), actor(r)); err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
		return
	case "disable":
		if err := service.DisableOpenVPNUser(r.Context(), bodyString(body, "username"), actor(r)); err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
		return
	case "getProps":
		props, err := service.GetOpenVPNUserProps(r.Context(), bodyString(body, "username"))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"props": props}})
		return
	case "setProps":
		if err := service.SetOpenVPNUserProps(r.Context(), bodyString(body, "username"), openVPNUserProps(body), actor(r), "", ""); err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
		return
	case "generateMFA":
		secret, err := service.GenerateOpenVPNUserMFA(r.Context(), bodyString(body, "username"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"totp_secret": secret}})
		return
	default:
		httputil.WriteError(w, domainerr.New(domainerr.CodeInvalidArgument, "invalid action"), requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
}

func (h *Handler) AdminOpenVPNGroupsAction(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	var body map[string]any
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	switch bodyString(body, "action") {
	case "create":
		group, err := service.CreateOpenVPNGroup(r.Context(), application.OpenVPNGroupCreateInput{
			Name:          bodyString(body, "name"),
			Description:   bodyString(body, "description"),
			PropAutologin: bodyBool(body, "prop_autologin"),
		}, actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true, "group": group}})
	case "update":
		group, err := service.UpdateOpenVPNGroup(r.Context(), bodyString(body, "groupname"), application.OpenVPNGroupUpdateInput{
			Description: optionalString(body, "description"),
		}, actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true, "group": group}})
	case "delete":
		err := service.DeleteOpenVPNGroup(r.Context(), bodyString(body, "groupname"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "getProps":
		props, err := service.GetOpenVPNGroupProps(r.Context(), bodyString(body, "groupname"))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"props": props}})
	case "setProps":
		err := service.SetOpenVPNGroupProps(r.Context(), bodyString(body, "groupname"), openVPNGroupProps(body), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "getMembers":
		members, err := service.GetOpenVPNGroupMembers(r.Context(), bodyString(body, "groupname"))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"members": members}})
	default:
		httputil.WriteError(w, domainerr.New(domainerr.CodeInvalidArgument, "invalid action"), requestID(r))
	}
}

func (h *Handler) AdminOpenVPNConnectionsAction(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	var body map[string]any
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	if bodyString(body, "action") != "disconnect" {
		httputil.WriteError(w, domainerr.New(domainerr.CodeInvalidArgument, "invalid action"), requestID(r))
		return
	}
	if err := service.DisconnectOpenVPNConnection(r.Context(), bodyString(body, "clientId"), actor(r)); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
}

func (h *Handler) AdminOpenVPNConfigsAction(w http.ResponseWriter, r *http.Request) {
	service, err := h.ensureAdminPortal()
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	var body map[string]any
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	switch bodyString(body, "action") {
	case "create":
		config, err := service.CreateOpenVPNConfig(r.Context(), application.OpenVPNConfigCreateInput{
			Username: bodyString(body, "username"),
			Name:     bodyString(body, "name"),
		}, actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true, "config": config}})
	case "download":
		download, err := service.DownloadOpenVPNConfig(r.Context(), bodyString(body, "configId"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"filename": download.Filename, "contentType": download.ContentType, "content": download.Content}})
	case "delete":
		err := service.DeleteOpenVPNConfig(r.Context(), bodyString(body, "configId"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "revoke":
		err := service.RevokeOpenVPNConfig(r.Context(), bodyString(body, "configId"), actor(r))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"success": true}})
	case "qrcode":
		qrCode, err := service.GetOpenVPNConfigQRCode(r.Context(), bodyString(body, "configId"))
		if err != nil {
			httputil.WriteError(w, err, requestID(r))
			return
		}
		httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: map[string]any{"qrCode": qrCode}})
	default:
		httputil.WriteError(w, domainerr.New(domainerr.CodeInvalidArgument, "invalid action"), requestID(r))
	}
}

func optionalString(data map[string]any, key string) *string {
	value, ok := data[key]
	if !ok || value == nil {
		return nil
	}
	text, ok := value.(string)
	if !ok {
		return nil
	}
	text = strings.TrimSpace(text)
	return &text
}

func optionalBool(data map[string]any, key string) *bool {
	value, ok := data[key]
	if !ok || value == nil {
		return nil
	}
	v, ok := value.(bool)
	if !ok {
		return nil
	}
	return &v
}

func openVPNUserProps(data map[string]any) application.OpenVPNUserProps {
	props := application.OpenVPNUserProps{}
	if raw, ok := data["props"].(map[string]any); ok {
		props.PropAutologin = bodyBool(raw, "prop_autologin")
		props.PropAdmin = bodyBool(raw, "prop_admin")
		props.PropDeny = bodyBool(raw, "prop_deny")
		props.PropAutogenerate = bodyBool(raw, "prop_autogenerate")
		if value, exists := raw["group"]; exists {
			if value == nil {
				empty := ""
				props.Group = &empty
			} else {
				props.Group = optionalString(raw, "group")
			}
		}
	}
	return props
}

func openVPNGroupProps(data map[string]any) application.OpenVPNGroupProps {
	props := application.OpenVPNGroupProps{}
	if raw, ok := data["props"].(map[string]any); ok {
		props.PropAutologin = bodyBool(raw, "prop_autologin")
		props.PropDeny = bodyBool(raw, "prop_deny")
	}
	return props
}
