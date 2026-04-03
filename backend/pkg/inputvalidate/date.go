package inputvalidate

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var ddmmyyyyPattern = regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`)

func NormalizeDDMMYYYY(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if !ddmmyyyyPattern.MatchString(trimmed) {
		return "", fmt.Errorf("must match dd/MM/yyyy")
	}
	parsed, err := time.Parse("02/01/2006", trimmed)
	if err != nil {
		return "", fmt.Errorf("must be a valid calendar date in dd/MM/yyyy")
	}
	if parsed.Format("02/01/2006") != trimmed {
		return "", fmt.Errorf("must match dd/MM/yyyy")
	}
	return trimmed, nil
}
