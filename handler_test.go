package formatter

import (
	"log/slog"
	"testing"
	"time"

	"github.com/mattn/go-colorable"
)

func TestHandle(t *testing.T) {
	handler := NewHandler(colorable.NewColorableStdout(), &Handler{
		DisableColors:            false,
		ForceColors:              true,
		DisableTimestamp:         false,
		UseUppercaseLevel:        false,
		UseTimePassedAsTimestamp: false,
		TimestampFormat:          time.StampMilli,
		PadAllLogEntries:         true,
		Level:                    slog.LevelDebug,
	})

	logger := slog.New(handler)

	testFieldMatches := map[string]string{
		"brownish": "232:94",
		"blue":     "232:33",
		"orange":   "232:172",
		"green":    "232:34",
	}

	for color, colorCode := range testFieldMatches {
		AddFieldMatchColorScheme("color", &FieldMatch{
			Value: color,
			Color: colorCode,
		})
		_ = colorCode
	}

	for color, colorCode := range testFieldMatches {
		logger.Info(
			"this message should have colorCode: "+colorCode+" (key: "+color+")",
			"color", color,
		)
	}

	logger.Info("this message should have no color", "color", "notRegisteredColor")

	// testing access the publicly registered field matches
	for _, colorSchemes := range FieldMatchColorScheme {
		for _, colorScheme := range colorSchemes {
			AddFieldMatchColorScheme("copiedColor", colorScheme)
		}
	}

	for color, colorCode := range testFieldMatches {
		for copiedColor, copiedColorCode := range testFieldMatches {
			logger.Info(
				"multi module (["+colorCode+"]["+copiedColorCode+"] - "+color+" & "+copiedColor+")",
				"color", color,
				"copiedColor", copiedColor,
			)
		}
	}
}

// TestWithAttrs tests that WithAttrs precomputes fields correctly
func TestWithAttrs(t *testing.T) {
	handler := NewHandler(colorable.NewColorableStdout(), &Handler{
		ForceColors:      true,
		TimestampFormat:  time.StampMilli,
		PadAllLogEntries: true,
		Level:            slog.LevelDebug,
	})

	AddFieldMatchColorScheme("module", &FieldMatch{
		Value: "test.com",
		Color: "232:34",
	})

	// simulate how watcher-go uses log.WithField("module", key)
	logger := slog.New(handler).With("module", "test.com")

	logger.Info("downloading updates")
	logger.Warn("rate limit approaching")
	logger.Error("connection failed")
	logger.Debug("opening GET uri")
}
