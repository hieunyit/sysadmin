package application

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	domainerr "backend/internal/domain/errors"
)

func validatePayload(v *validator.Validate, payload any, message string) error {
	if v == nil {
		return nil
	}
	if err := v.Struct(payload); err != nil {
		var validationErrs validator.ValidationErrors
		if ok := errorsAsValidation(err, &validationErrs); ok {
			return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, message, validationErrorsToDetails(payload, validationErrs))
		}
		return domainerr.Wrap(domainerr.CodeInvalidArgument, message, err)
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
