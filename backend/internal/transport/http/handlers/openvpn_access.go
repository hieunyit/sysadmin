package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	domainerr "backend/internal/domain/errors"
	"backend/internal/domain/services"
	"backend/pkg/httputil"
)

func (h *Handler) AppendUserAccessList(w http.ResponseWriter, r *http.Request) {
	h.handleAccessListMutationForOwner(w, r, "append", "user", chi.URLParam(r, "username"))
}

func (h *Handler) RemoveUserAccessList(w http.ResponseWriter, r *http.Request) {
	h.handleAccessListMutationForOwner(w, r, "remove", "user", chi.URLParam(r, "username"))
}

func (h *Handler) AppendGroupAccessList(w http.ResponseWriter, r *http.Request) {
	h.handleAccessListMutationForOwner(w, r, "append", "group", chi.URLParam(r, "groupname"))
}

func (h *Handler) RemoveGroupAccessList(w http.ResponseWriter, r *http.Request) {
	h.handleAccessListMutationForOwner(w, r, "remove", "group", chi.URLParam(r, "groupname"))
}

func (h *Handler) ListUserOpenVPNAccessLists(w http.ResponseWriter, r *http.Request) {
	h.listOpenVPNAccessListsForOwner(w, r, "user", chi.URLParam(r, "username"))
}

func (h *Handler) ListGroupOpenVPNAccessLists(w http.ResponseWriter, r *http.Request) {
	h.listOpenVPNAccessListsForOwner(w, r, "group", chi.URLParam(r, "groupname"))
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
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: out})
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
			"keycloak_user_id":  result.UsedKeycloakID,
			"keycloak_username": strings.TrimSpace(result.User.Username),
			"keycloak_email":    strings.TrimSpace(result.User.Email),
			"openvpn_username":  result.OpenVPNUser,
			"openvpn_group":     result.OpenVPNGroup,
		},
		Warnings: result.Warnings,
	})
}

func (h *Handler) listOpenVPNAccessListsForOwner(w http.ResponseWriter, r *http.Request, subjectType, subject string) {
	subject, err := normalizeAccessListOwner(subjectType, subject)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}

	query := services.OpenVPNAccessListQuery{SubjectType: subjectType}
	switch subjectType {
	case "user":
		query.Username = subject
	case "group":
		query.Groupname = subject
	default:
		httputil.WriteError(w, domainerr.New(domainerr.CodeInvalidArgument, "unsupported subject type"), requestID(r))
		return
	}

	out, err := h.services.OpenVPNAdmin.ListAccessEntries(r.Context(), query)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: out})
}

func (h *Handler) handleAccessListMutationForOwner(w http.ResponseWriter, r *http.Request, mode, subjectType, subject string) {
	subject, err := normalizeAccessListOwner(subjectType, subject)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}

	var req accessListDTO
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	if err := validateStruct(h.validate, req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	items, err := mapAccessEntriesForOwner(req.Items, subjectType, subject)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	if err := h.services.OpenVPNAdmin.ApplyAccessEntries(r.Context(), mode, items, actor(r)); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func normalizeAccessListOwner(subjectType, subject string) (string, error) {
	subject = strings.TrimSpace(subject)
	field := "username"
	if subjectType == "group" {
		field = "groupname"
	}
	if subject == "" {
		return "", domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
			field: "is required",
		})
	}
	return subject, nil
}

func mapAccessEntriesForOwner(items []accessRouteDTO, subjectType, subject string) ([]services.OpenVPNAccessEntryInput, error) {
	mapped := mapAccessEntries(items)
	for i := range mapped {
		username := strings.TrimSpace(pointerString(mapped[i].Username))
		groupname := strings.TrimSpace(pointerString(mapped[i].Groupname))
		switch subjectType {
		case "user":
			if groupname != "" {
				return nil, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
					"items[" + strconv.Itoa(i) + "].groupname": "is not allowed for user access-list endpoint",
				})
			}
			if username != "" && !strings.EqualFold(username, subject) {
				return nil, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
					"items[" + strconv.Itoa(i) + "].username": "must match path username",
				})
			}
			owner := subject
			mapped[i].Username = &owner
			mapped[i].Groupname = nil
		case "group":
			if username != "" {
				return nil, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
					"items[" + strconv.Itoa(i) + "].username": "is not allowed for group access-list endpoint",
				})
			}
			if groupname != "" && !strings.EqualFold(groupname, subject) {
				return nil, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
					"items[" + strconv.Itoa(i) + "].groupname": "must match path groupname",
				})
			}
			owner := subject
			mapped[i].Groupname = &owner
			mapped[i].Username = nil
		default:
			return nil, domainerr.New(domainerr.CodeInvalidArgument, "unsupported subject type")
		}
	}
	return mapped, nil
}

func pointerString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
