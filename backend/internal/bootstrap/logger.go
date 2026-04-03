package bootstrap

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
)

func NewLogger(level string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return logger
}
