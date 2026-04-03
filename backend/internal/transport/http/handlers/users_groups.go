package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	appcmd "backend/internal/application/commands"
	"backend/internal/domain/services"
	"backend/pkg/httputil"
)

type userSummaryDTO struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Enabled     bool     `json:"enabled"`
	Groups      []string `json:"groups"`
}

func toUserSummaryDTO(u services.KeycloakUser) userSummaryDTO {
	return userSummaryDTO{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Enabled:     u.Enabled,
		Groups:      u.Groups,
	}
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserDTO
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	if err := validateStruct(h.validate, req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	firstName := req.FirstName
	if firstName == "" {
		firstName = req.FirstNameAlt
	}
	lastName := req.LastName
	if lastName == "" {
		lastName = req.LastNameAlt
	}
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.DisplayNameAlt
	}
	notificationEmail := req.NotificationEmail
	if notificationEmail == "" {
		notificationEmail = req.NotificationEmailAlt
	}
	identitySource := req.IdentitySource
	if identitySource == "" {
		identitySource = req.IdentitySourceAlt
	}
	if displayName == "" && req.Attributes != nil {
		if fullName, ok := req.Attributes["fullName"]; ok {
			displayName = fullName
		}
	}
	attributes := cloneStringMap(req.Attributes)
	if attributes == nil {
		attributes = make(map[string]string)
	}
	if onboardDate := firstNonEmpty(req.OnboardDate, req.OnboardDateAlt, req.OnboardLegacy); onboardDate != "" {
		attributes["onboardDate"] = onboardDate
	}
	if workAddress := firstNonEmpty(req.WorkAddress, req.WorkAddressAlt, req.AddressLegacy); workAddress != "" {
		attributes["workAddress"] = workAddress
	}

	requiredActions := req.RequiredActions
	if len(requiredActions) == 0 && len(req.RequiredActionsAlt) > 0 {
		requiredActions = req.RequiredActionsAlt
	}

	emailVerified := true
	if req.EmailVerified != nil {
		emailVerified = *req.EmailVerified
	} else if req.EmailVerifiedAlt != nil {
		emailVerified = *req.EmailVerifiedAlt
	}

	passwordTemporary := false
	if req.PasswordTemporary != nil {
		passwordTemporary = *req.PasswordTemporary
	} else if req.PasswordTemporaryAlt != nil {
		passwordTemporary = *req.PasswordTemporaryAlt
	}

	cmd := appcmd.CreateUser{
		Username:          req.Username,
		Email:             req.Email,
		NotificationEmail: notificationEmail,
		FirstName:         firstName,
		LastName:          lastName,
		DisplayName:       displayName,
		IdentitySource:    identitySource,
		EmailVerified:     emailVerified,
		RequiredActions:   requiredActions,
		Attributes:        attributes,
		Groups:            req.Groups,
		Enabled:           req.Enabled,
		Password:          req.Password,
		PasswordTemporary: passwordTemporary,
		Actor:             actor(r),
	}

	result, err := h.services.User.Create(r.Context(), cmd)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, httputil.SuccessResponse{
		Data:     toUserSummaryDTO(result.User),
		Warnings: result.Warnings,
	})
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstStringPointer(values ...*string) *string {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstBoolPointer(values ...*bool) *bool {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstStringSlicePointer(values ...*[]string) *[]string {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func mergeWarnings(current []string, warnings ...string) []string {
	seen := make(map[string]struct{}, len(current))
	out := make([]string, 0, len(current)+len(warnings))
	for _, warning := range append(current, warnings...) {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}
		if _, ok := seen[warning]; ok {
			continue
		}
		seen[warning] = struct{}{}
		out = append(out, warning)
	}
	return out
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit, offset, err := parseLimitOffset(r, 50)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	users, err := h.services.User.List(r.Context(), q, offset, limit)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	items := make([]userSummaryDTO, 0, len(users))
	for _, u := range users {
		items = append(items, toUserSummaryDTO(u))
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{
		Data: items,
	})
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := h.services.User.Get(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	dto := toUserSummaryDTO(u)
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{
		Data: dto,
	})
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateUserDTO
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	if err := validateStruct(h.validate, req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	attrs := req.Attributes
	if attrs != nil {
		cloned := cloneStringMap(*attrs)
		attrs = &cloned
	}
	u, err := h.services.User.Update(r.Context(), id, appcmd.UpdateUser{
		Username:        req.Username,
		Email:           req.Email,
		FirstName:       firstStringPointer(req.FirstName, req.FirstNameAlt),
		LastName:        firstStringPointer(req.LastName, req.LastNameAlt),
		DisplayName:     firstStringPointer(req.DisplayName, req.DisplayNameAlt),
		Enabled:         req.Enabled,
		EmailVerified:   firstBoolPointer(req.EmailVerified, req.EmailVerifiedAlt),
		RequiredActions: firstStringSlicePointer(req.RequiredActions, req.RequiredActionsAlt),
		Attributes:      attrs,
		Actor:           actor(r),
	})
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	dto := toUserSummaryDTO(u)
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{
		Data: dto,
	})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.services.User.Delete(r.Context(), id); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req createGroupDTO
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	if err := validateStruct(h.validate, req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	group, err := h.services.Group.Create(r.Context(), appcmd.CreateGroup{Name: req.Name, Actor: actor(r)})
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, httputil.SuccessResponse{Data: group})
}

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	groups, err := h.services.Group.List(r.Context(), q)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: groups})
}

func (h *Handler) GetGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	g, err := h.services.Group.Get(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: g})
}

func (h *Handler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateGroupDTO
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	if err := validateStruct(h.validate, req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	g, err := h.services.Group.Update(r.Context(), id, appcmd.UpdateGroup{Name: req.Name, Actor: actor(r)})
	if err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, httputil.SuccessResponse{Data: g})
}

func (h *Handler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.services.Group.Delete(r.Context(), id); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req addMemberDTO
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	if err := validateStruct(h.validate, req); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	if err := h.services.Group.AddMember(r.Context(), id, appcmd.AddGroupMember{UserID: req.UserID, Actor: actor(r)}); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")
	if err := h.services.Group.RemoveMember(r.Context(), groupID, userID); err != nil {
		httputil.WriteError(w, err, requestID(r))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
