package handlers

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"

	"backend/internal/application"
	domainerr "backend/internal/domain/errors"
	"backend/internal/domain/services"
	mw "backend/internal/transport/http/middleware"
	"backend/pkg/httputil"
)

type Handler struct {
	services *application.Services
	validate *validator.Validate
}

func New(services *application.Services, validate *validator.Validate) *Handler {
	return &Handler{services: services, validate: validate}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.Health)
	r.Get("/ready", h.Ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/keycloak", func(r chi.Router) {
			r.Post("/users", h.CreateUser)
			r.Get("/users", h.ListUsers)
			r.Get("/users/{id}", h.GetUser)
			r.Put("/users/{id}", h.UpdateUser)
			r.Patch("/users/{id}", h.UpdateUser)
			r.Delete("/users/{id}", h.DeleteUser)

			r.Post("/groups", h.CreateGroup)
			r.Get("/groups", h.ListGroups)
			r.Get("/groups/{id}", h.GetGroup)
			r.Put("/groups/{id}", h.UpdateGroup)
			r.Patch("/groups/{id}", h.UpdateGroup)
			r.Delete("/groups/{id}", h.DeleteGroup)

			r.Post("/groups/{id}/members", h.AddMember)
			r.Delete("/groups/{id}/members/{userId}", h.RemoveMember)
		})

		r.Post("/openvpn/users:create-from-keycloak", h.CreateOpenVPNUserFromKeycloak)
		r.Get("/openvpn/users", h.ListOpenVPNUsers)
		r.Get("/openvpn/users:export", h.ExportOpenVPNUsers)
		r.Get("/openvpn/groups", h.ListOpenVPNGroups)

		r.Get("/openvpn/users/{username}/access-lists", h.ListUserOpenVPNAccessLists)
		r.Post("/openvpn/users/{username}/access-lists:append", h.AppendUserAccessList)
		r.Post("/openvpn/users/{username}/access-lists:remove", h.RemoveUserAccessList)
		r.Get("/openvpn/groups/{groupname}/access-lists", h.ListGroupOpenVPNAccessLists)
		r.Post("/openvpn/groups/{groupname}/access-lists:append", h.AppendGroupAccessList)
		r.Post("/openvpn/groups/{groupname}/access-lists:remove", h.RemoveGroupAccessList)
	})
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) Ready(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"status": "ready"})
}

func requestID(r *http.Request) string {
	id := mw.RequestID(r.Context())
	if id != "" {
		return id
	}
	return chimw.GetReqID(r.Context())
}

func actor(r *http.Request) string {
	return mw.Actor(r.Context())
}

const maxPageSize = 500

func parseLimitOffset(r *http.Request, defLimit int) (int, int, error) {
	limit := defLimit
	offset := 0
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, 0, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"limit": "must be a positive integer",
			})
		}
		if i <= 0 {
			return 0, 0, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"limit": "must be greater than 0",
			})
		}
		if i > maxPageSize {
			return 0, 0, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"limit": fmt.Sprintf("must be less than or equal to %d", maxPageSize),
			})
		}
		limit = i
	}
	if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, 0, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"offset": "must be a non-negative integer",
			})
		}
		if i < 0 {
			return 0, 0, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
				"offset": "must be greater than or equal to 0",
			})
		}
		offset = i
	}
	return limit, offset, nil
}

func parseOptionalBoolQuery(r *http.Request, name string, def bool) (bool, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return def, nil
	}
	switch strings.ToLower(raw) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return def, domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
			name: "must be true or false",
		})
	}
}

func validateStruct(v *validator.Validate, payload any) error {
	if v == nil {
		return nil
	}
	if err := v.Struct(payload); err != nil {
		var validationErrs validator.ValidationErrors
		if ok := errorsAsValidation(err, &validationErrs); ok {
			return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", validationErrorsToDetails(payload, validationErrs))
		}
		return domainerr.Wrap(domainerr.CodeInvalidArgument, "validation failed", err)
	}
	return nil
}

func errorsAsValidation(err error, target *validator.ValidationErrors) bool {
	if err == nil || target == nil {
		return false
	}
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return false
	}
	*target = ve
	return true
}

func validationErrorsToDetails(payload any, errs validator.ValidationErrors) map[string]string {
	if len(errs) == 0 {
		return nil
	}
	details := make(map[string]string, len(errs))
	for _, verr := range errs {
		details[jsonFieldName(payload, verr.StructField(), verr.Field())] = validationMessage(verr)
	}
	return details
}

func jsonFieldName(payload any, structFieldName, fallback string) string {
	typ := reflect.TypeOf(payload)
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ != nil && typ.Kind() == reflect.Struct {
		if field, ok := typ.FieldByName(structFieldName); ok {
			tag := strings.Split(field.Tag.Get("json"), ",")[0]
			if tag != "" && tag != "-" {
				return tag
			}
		}
	}
	if fallback == "" {
		return "field"
	}
	return strings.ToLower(fallback[:1]) + fallback[1:]
}

func validationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		switch err.Kind() {
		case reflect.String, reflect.Slice, reflect.Array:
			return fmt.Sprintf("must have minimum length %s", err.Param())
		default:
			return fmt.Sprintf("must be at least %s", err.Param())
		}
	case "max":
		switch err.Kind() {
		case reflect.String, reflect.Slice, reflect.Array:
			return fmt.Sprintf("must have maximum length %s", err.Param())
		default:
			return fmt.Sprintf("must be at most %s", err.Param())
		}
	case "oneof":
		return fmt.Sprintf("must be one of: %s", strings.ReplaceAll(err.Param(), " ", ", "))
	default:
		return fmt.Sprintf("failed validation: %s", err.Tag())
	}
}

func mapAccessEntries(items []accessRouteDTO) []services.OpenVPNAccessEntryInput {
	out := make([]services.OpenVPNAccessEntryInput, 0, len(items))
	for _, it := range items {
		out = append(out, services.OpenVPNAccessEntryInput{
			Username:    it.Username,
			Groupname:   it.Groupname,
			Target:      it.Target,
			Type:        it.Type,
			RouteType:   it.RouteType,
			Accept:      it.Accept,
			CIDR:        it.CIDR,
			ServiceSpec: it.ServiceSpec,
			Domain:      it.Domain,
			MatchType:   it.MatchType,
			Action:      it.Action,
			Position:    it.Position,
			Comment:     it.Comment,
		})
	}
	return out
}
