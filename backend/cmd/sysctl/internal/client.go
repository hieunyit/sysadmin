package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

type APIError struct {
	Method    string
	Path      string
	Status    int
	Code      string
	Message   string
	RequestID string
	Details   map[string]string
	RawBody   string
}

func (e *APIError) Error() string {
	var message string
	if friendly := e.friendlyMessage(); friendly != "" {
		message = friendly
	} else {
		switch {
		case e.Code != "" && e.Message != "":
			message = fmt.Sprintf("%s %s failed (%d %s): %s", e.Method, e.Path, e.Status, e.Code, e.Message)
		case e.Message != "":
			message = fmt.Sprintf("%s %s failed (%d): %s", e.Method, e.Path, e.Status, e.Message)
		case strings.TrimSpace(e.RawBody) != "":
			message = fmt.Sprintf("%s %s failed (%d): %s", e.Method, e.Path, e.Status, strings.TrimSpace(e.RawBody))
		default:
			message = fmt.Sprintf("%s %s failed (%d)", e.Method, e.Path, e.Status)
		}
	}
	if len(e.Details) > 0 {
		keys := make([]string, 0, len(e.Details))
		for key := range e.Details {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		var b strings.Builder
		b.WriteString(message)
		b.WriteString("\n\nDetails:")
		for _, key := range keys {
			b.WriteString("\n- ")
			b.WriteString(key)
			b.WriteString(": ")
			b.WriteString(e.Details[key])
		}
		message = b.String()
	}
	if strings.TrimSpace(e.RequestID) != "" {
		message += "\n\nRequest ID: " + strings.TrimSpace(e.RequestID)
	}
	return message
}

func (e *APIError) friendlyMessage() string {
	if e == nil {
		return ""
	}
	if friendly := e.friendlyUserCreateMessage(); friendly != "" {
		return friendly
	}
	if e.Code != "precondition_failed" || !strings.Contains(e.Path, "/api/v1/openvpn/access-lists:") {
		return ""
	}
	if !strings.Contains(strings.ToLower(e.Message), "has no openvpn ruleset for domain routing") {
		return ""
	}

	groupName := extractRulesetGroupName(e.Message)
	subject := "This group"
	if groupName != "" {
		subject = fmt.Sprintf("Group %q", groupName)
	}

	return fmt.Sprintf(
		"%s %s failed (%d %s): %s\n\nHow to fix:\n1. Create a domain-routing ruleset for %s in OpenVPN AS.\n2. Assign that ruleset to the same group.\n3. Run the same access-list command again.\n\nNotes:\n- Group IP/CIDR entries do not need a ruleset.\n- User domain entries can auto-bootstrap a ruleset; group domain entries cannot.",
		e.Method,
		e.Path,
		e.Status,
		e.Code,
		e.Message,
		subject,
	)
}

func (e *APIError) friendlyUserCreateMessage() string {
	if e == nil {
		return ""
	}
	if e.Code != "precondition_failed" || !strings.Contains(e.Path, "/api/v1/keycloak/users") {
		return ""
	}

	switch {
	case strings.Contains(e.Message, "requires KEYCLOAK_LDAP_COMPONENT_ID"):
		return fmt.Sprintf(
			"%s %s failed (%d %s): %s\n\nHow to fix:\n1. Open the LDAP provider in Keycloak admin.\n2. Copy the component ID from the URL ending with /components/<id>.\n3. Set KEYCLOAK_LDAP_COMPONENT_ID in the backend env.\n4. Restart the backend and run the command again.",
			e.Method, e.Path, e.Status, e.Code, e.Message,
		)
	case strings.Contains(e.Message, "requires LDAP provider editMode to allow writes"):
		return fmt.Sprintf(
			"%s %s failed (%d %s): %s\n\nHow to fix:\n1. Open the LDAP provider in Keycloak admin.\n2. Change editMode from READ_ONLY to a write-capable mode such as WRITABLE.\n3. Keep using create-ldap for LDAP-backed users.\n4. Run the command again.",
			e.Method, e.Path, e.Status, e.Code, e.Message,
		)
	}

	return ""
}

var groupNoRulesetPattern = regexp.MustCompile(`group\s+"([^"]+)"\s+has no OpenVPN ruleset for domain routing`)

func extractRulesetGroupName(message string) string {
	matches := groupNoRulesetPattern.FindStringSubmatch(strings.TrimSpace(message))
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) Do(ctx context.Context, method, path string, payload any) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, _ := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var body struct {
			Error struct {
				Code      string            `json:"code"`
				Message   string            `json:"message"`
				RequestID string            `json:"request_id"`
				Details   map[string]string `json:"details"`
			} `json:"error"`
		}
		if err := json.Unmarshal(b, &body); err == nil && (body.Error.Code != "" || body.Error.Message != "") {
			return b, resp.StatusCode, &APIError{
				Method:    method,
				Path:      path,
				Status:    resp.StatusCode,
				Code:      body.Error.Code,
				Message:   body.Error.Message,
				RequestID: body.Error.RequestID,
				Details:   body.Error.Details,
				RawBody:   string(b),
			}
		}
		return b, resp.StatusCode, &APIError{
			Method:  method,
			Path:    path,
			Status:  resp.StatusCode,
			RawBody: string(b),
		}
	}
	return b, resp.StatusCode, nil
}
