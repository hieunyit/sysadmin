package handlers

import (
	"net/http"
	"strings"

	domainerr "backend/internal/domain/errors"
	"backend/internal/domain/services"
	"backend/pkg/httputil"
)

func (h *Handler) AppendAccessList(w http.ResponseWriter, r *http.Request) {
	h.handleAccessListMutation(w, r, "append")
}

func (h *Handler) RemoveAccessList(w http.ResponseWriter, r *http.Request) {
	h.handleAccessListMutation(w, r, "remove")
}

func (h *Handler) ListOpenVPNUsers(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit, offset, err := parseLimitOffset(r, 50)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	all, err := parseOptionalBoolQuery(r, "all", false)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	query := services.OpenVPNUserListQuery{
		Limit:  limit,
		Offset: offset,
		Search: q,
	}
	var out map[string]any
	if all {
		out, err = h.services.OpenVPNAdmin.ListAllUsers(r.Context(), query)
	} else {
		out, err = h.services.OpenVPNAdmin.ListUsers(r.Context(), query)
	}
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{
		Data:     out,
		Warnings: openVPNUserWarnings(out),
	})
}

func (h *Handler) ListOpenVPNGroups(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit, offset, err := parseLimitOffset(r, 50)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	enumerateMembers, err := parseOptionalBoolQuery(r, "enumerate_members", false)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	all, err := parseOptionalBoolQuery(r, "all", false)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	query := services.OpenVPNGroupListQuery{
		Limit:            limit,
		Offset:           offset,
		Search:           q,
		EnumerateMembers: enumerateMembers,
	}
	var out map[string]any
	if all {
		out, err = h.services.OpenVPNAdmin.ListAllGroups(r.Context(), query)
	} else {
		out, err = h.services.OpenVPNAdmin.ListGroups(r.Context(), query)
	}
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: out})
}

func (h *Handler) ExportOpenVPNUsers(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	rows, warnings, err := h.services.OpenVPNAdmin.ExportUsers(r.Context(), q)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{
		Data:     rows,
		Warnings: warnings,
	})
}

func (h *Handler) CreateOpenVPNUserFromKeycloak(w http.ResponseWriter, r *http.Request) {
	var req createOpenVPNFromKeycloakDTO
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	userID := firstNonEmpty(req.UserID, req.UserIDAlt)
	username := strings.TrimSpace(req.Username)
	vpnGroup := strings.TrimSpace(req.VPNGroup)

	if h.services.OpenVPNProvisioner == nil {
		httputil.WriteError(w, domainerr.New(domainerr.CodePreconditionFail, "openvpn provisioner is not configured"), requestID(r))
		return
	}

	result, err := h.services.OpenVPNProvisioner.CreateUserFromKeycloak(r.Context(), userID, username, vpnGroup)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, httputil.SuccessResponse{
		Data: map[string]any{
			"keycloak_user_id": result.UsedKeycloakID,
			"keycloak_username": strings.TrimSpace(result.User.Username),
			"keycloak_email": strings.TrimSpace(result.User.Email),
			"openvpn_username": result.OpenVPNUser,
			"openvpn_group": result.OpenVPNGroup,
		},
		Warnings: result.Warnings,
	})
}

func (h *Handler) ListOpenVPNAccessLists(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	groupname := strings.TrimSpace(r.URL.Query().Get("groupname"))
	subjectType := strings.TrimSpace(r.URL.Query().Get("subject_type"))
	if subjectType == "" {
		subjectType = strings.TrimSpace(r.URL.Query().Get("object_type"))
	}
	if username != "" && groupname != "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, string(domainerr.CodeInvalidArgument), "set either username or groupname", requestID(r), map[string]string{
			"username":  "cannot be combined with groupname",
			"groupname": "cannot be combined with username",
		})
		return
	}
	if subjectType == "" {
		switch {
		case username != "" && groupname == "":
			subjectType = "user"
		case groupname != "" && username == "":
			subjectType = "group"
		}
	}
	if subjectType != "" && subjectType != "user" && subjectType != "group" {
		httputil.WriteAPIError(w, http.StatusBadRequest, string(domainerr.CodeInvalidArgument), "subject_type must be user or group", requestID(r), map[string]string{
			"subject_type": "must be one of: user, group",
		})
		return
	}
	if username == "" && groupname == "" && subjectType == "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, string(domainerr.CodeInvalidArgument), "access-list is scoped to user/group; set username, groupname or subject_type", requestID(r), map[string]string{
			"subject": "set username or groupname",
		})
		return
	}
	if subjectType == "user" && username == "" && groupname != "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, string(domainerr.CodeInvalidArgument), "subject_type=user requires username filter", requestID(r), map[string]string{
			"username": "is required when subject_type=user",
		})
		return
	}
	if subjectType == "group" && groupname == "" && username != "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, string(domainerr.CodeInvalidArgument), "subject_type=group requires groupname filter", requestID(r), map[string]string{
			"groupname": "is required when subject_type=group",
		})
		return
	}

	out, err := h.services.OpenVPNAdmin.ListAccessEntries(r.Context(), services.OpenVPNAccessListQuery{
		Username:    username,
		Groupname:   groupname,
		SubjectType: subjectType,
	})
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: out})
}

func (h *Handler) handleAccessListMutation(w http.ResponseWriter, r *http.Request, mode string) {
	var req accessListDTO
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	if err := validateStruct(h.validate, req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	items := mapAccessEntries(req.Items)
	if err := h.services.OpenVPNAdmin.ApplyAccessEntries(r.Context(), mode, items, actor(r)); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func openVPNUserWarnings(out map[string]any) []string {
	profiles, ok := out["profiles"].([]any)
	if !ok {
		return nil
	}
	for _, item := range profiles {
		profile, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if failed, ok := profile["last_vpn_login_lookup_failed"].(bool); ok && failed {
			return []string{"Không thể lấy thời gian đăng nhập VPN gần nhất từ Keycloak events cho một số người dùng; dữ liệu last_vpn_login_at có thể chưa đầy đủ."}
		}
	}
	return nil
}
