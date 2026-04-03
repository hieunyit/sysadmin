package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	domainerr "backend/internal/domain/errors"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	RequestID string            `json:"request_id,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

type SuccessResponse struct {
	Data     any      `json:"data"`
	Warnings []string `json:"warnings,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, err error, requestID string) {
	status := http.StatusInternalServerError
	code := string(domainerr.CodeInternal)
	message := "internal error"
	var details map[string]string

	if de, ok := err.(*domainerr.DomainError); ok {
		code = string(de.Code)
		message = de.Message
		details = cloneDetails(de.Details)
		if de.Cause != nil {
			switch de.Code {
			case domainerr.CodeInvalidArgument:
				if len(details) == 0 {
					details = map[string]string{}
				}
				if _, exists := details["cause"]; !exists {
					details["cause"] = de.Cause.Error()
				}
			}
		}
		switch de.Code {
		case domainerr.CodeInvalidArgument:
			status = http.StatusBadRequest
		case domainerr.CodeNotFound:
			status = http.StatusNotFound
		case domainerr.CodeConflict:
			status = http.StatusConflict
		case domainerr.CodeUnauthorized:
			status = http.StatusUnauthorized
		case domainerr.CodeForbidden:
			status = http.StatusForbidden
		case domainerr.CodePreconditionFail:
			status = http.StatusPreconditionFailed
		case domainerr.CodeExternalFailure:
			status = http.StatusBadGateway
		default:
			status = http.StatusInternalServerError
		}
	}

	WriteAPIError(w, status, code, message, requestID, details)
}

func DecodeJSON(r *http.Request, out any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(out); err != nil {
		return decodeJSONError(err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "invalid JSON request body", map[string]string{
			"body": "must contain a single JSON object",
		})
	}
	return nil
}

var unknownFieldPattern = regexp.MustCompile(`json: unknown field "([^"]+)"`)

func decodeJSONError(err error) error {
	switch {
	case errors.Is(err, io.EOF):
		return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "invalid JSON request body", map[string]string{
			"body": "request body is required",
		})
	}

	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "invalid JSON request body", map[string]string{
			"body": fmt.Sprintf("must not exceed %d bytes", maxBytesErr.Limit),
		})
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "invalid JSON request body", map[string]string{
			"body": fmt.Sprintf("invalid JSON syntax at byte offset %d", syntaxErr.Offset),
		})
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := strings.TrimSpace(typeErr.Field)
		if field == "" {
			field = "body"
		}
		return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "invalid JSON request body", map[string]string{
			field: fmt.Sprintf("must be %s", typeErr.Type.String()),
		})
	}

	if matches := unknownFieldPattern.FindStringSubmatch(err.Error()); len(matches) == 2 {
		return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "invalid JSON request body", map[string]string{
			matches[1]: "is not allowed",
		})
	}

	return domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "invalid JSON request body", map[string]string{
		"body": err.Error(),
	})
}

func WriteAPIError(w http.ResponseWriter, status int, code, message, requestID string, details map[string]string) {
	WriteJSON(w, status, ErrorResponse{
		Error: ErrorBody{
			Code:      code,
			Message:   message,
			RequestID: requestID,
			Details:   cloneDetails(details),
		},
	})
}

func cloneDetails(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
