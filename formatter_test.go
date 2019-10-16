package formatter

import (
	"testing"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/sirupsen/logrus"
)

func TestFormat(t *testing.T) {
	lvl, _ := logrus.ParseLevel("debug")
	logrus.SetLevel(lvl)
	// set custom text formatter for the logger
	logrus.StandardLogger().Formatter = &Formatter{
		DisableColors:            false,
		ForceColors:              true,
		DisableTimestamp:         false,
		UseUppercaseLevel:        false,
		UseTimePassedAsTimestamp: false,
		TimestampFormat:          time.StampMilli,
		PadAllLogEntries:         true,
	}
	logrus.SetOutput(colorable.NewColorableStdout())

	testFieldMatches := map[string]string{
		"brownish": "232:94",
		"blue":     "232:33",
		"orange":   "232:172",
		"green":    "232:34",
	}

	for color, codeCode := range testFieldMatches {
		AddFieldMatchColorScheme("color", &FieldMatch{
			Value: color,
			Color: codeCode,
		})
	}

	for color, colorCode := range testFieldMatches {
		logrus.WithField("color", color).Infof(
			"this message should have colorCode: %s (key: %s)", colorCode, color,
		)
	}

	logrus.WithField("color", "notRegisteredColor").Info("this message should have no color")

	// testing access the publicly registered field matches
	for _, colorSchemes := range FieldMatchColorScheme {
		for _, colorScheme := range colorSchemes {
			AddFieldMatchColorScheme("copiedColor", colorScheme)
		}
	}

	for color, colorCode := range testFieldMatches {
		for copiedColor, copiedColorCode := range testFieldMatches {
			logrus.WithFields(logrus.Fields{"color": color, "copiedColor": copiedColor}).Infof(
				"multi module ([%s][%s] - %s & %s)", colorCode, copiedColorCode, color, copiedColor,
			)
		}
	}
}
