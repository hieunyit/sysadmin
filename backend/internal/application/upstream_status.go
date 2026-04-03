package application

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

var upstreamStatusPattern = regexp.MustCompile(`status=(\d{3})`)
var upstreamBodyPattern = regexp.MustCompile(`body=(.+)$`)

func upstreamStatusCode(err error) int {
	if err == nil {
		return 0
	}
	matches := upstreamStatusPattern.FindStringSubmatch(err.Error())
	if len(matches) != 2 {
		return 0
	}
	code, convErr := strconv.Atoi(matches[1])
	if convErr != nil {
		return 0
	}
	return code
}

func isUpstreamStatus(err error, code int) bool {
	return upstreamStatusCode(err) == code
}

func upstreamBody(err error) string {
	if err == nil {
		return ""
	}
	matches := upstreamBodyPattern.FindStringSubmatch(strings.TrimSpace(err.Error()))
	if len(matches) != 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func upstreamSummary(err error) string {
	body := upstreamBody(err)
	if body == "" {
		return ""
	}

	var payload map[string]any
	if json.Unmarshal([]byte(body), &payload) == nil {
		for _, key := range []string{"errorMessage", "error_description", "message", "error"} {
			if value := strings.TrimSpace(stringValue(payload[key])); value != "" {
				return value
			}
		}
	}

	return body
}

func stringValue(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}
