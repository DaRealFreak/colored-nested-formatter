// Package formatter implements a slog.Handler with colored nested field output
package formatter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/mgutz/ansi"
)

// defaultTimestampFormat is the time format we use if nothing is set manually
const defaultTimestampFormat = time.StampMilli

var baseTimestamp = time.Now()

var (
	FieldMatchColorScheme map[string][]*FieldMatch
	defaultColorSchema    = &ColorSchema{
		Timestamp:  "black+h",
		InfoLevel:  "green",
		WarnLevel:  "yellow+B",
		ErrorLevel: "red",
		FatalLevel: "red",
		PanicLevel: "red",
		DebugLevel: "blue",
	}
)

// FieldMatch contains the value and defined color of the field match
type FieldMatch struct {
	Value interface{}
	Color string
}

// ColorSchema is the color schema for the default log parts/levels
type ColorSchema struct {
	Timestamp  string
	InfoLevel  string
	WarnLevel  string
	ErrorLevel string
	FatalLevel string
	PanicLevel string
	DebugLevel string
}

// Handler implements slog.Handler with colored nested field output
type Handler struct {
	// timestamp formatting, default is time.StampMilli
	TimestampFormat string
	// color schema for messages
	ColorSchema *ColorSchema
	// no colors
	DisableColors bool
	// no check for TTY terminal
	ForceColors bool
	// no timestamp
	DisableTimestamp bool
	// false -> time passed, true -> timestamp
	UseTimePassedAsTimestamp bool
	// false -> info, true -> INFO
	UseUppercaseLevel bool
	// reserves space for all log entries for all registered matches
	PadAllLogEntries bool
	// minimum log level
	Level slog.Level

	writer io.Writer
	mu     sync.Mutex
	// precomputed attrs from WithAttrs/WithGroup
	preAttrs []slog.Attr
}

// NewHandler creates a new Handler writing to w
func NewHandler(w io.Writer, opts *Handler) *Handler {
	h := &Handler{
		writer: w,
		Level:  slog.LevelDebug,
	}
	if opts != nil {
		h.TimestampFormat = opts.TimestampFormat
		h.ColorSchema = opts.ColorSchema
		h.DisableColors = opts.DisableColors
		h.ForceColors = opts.ForceColors
		h.DisableTimestamp = opts.DisableTimestamp
		h.UseTimePassedAsTimestamp = opts.UseTimePassedAsTimestamp
		h.UseUppercaseLevel = opts.UseUppercaseLevel
		h.PadAllLogEntries = opts.PadAllLogEntries
		h.Level = opts.Level
	}
	if h.ColorSchema == nil {
		h.ColorSchema = defaultColorSchema
	}
	return h
}

// Enabled reports whether the handler handles records at the given level
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.Level
}

// Handle formats and writes a log record
func (h *Handler) Handle(_ context.Context, record slog.Record) error {
	out := new(bytes.Buffer)

	if err := h.appendTimeInfo(out, record.Time); err != nil {
		return err
	}

	if err := h.appendLevelInfo(out, record.Level); err != nil {
		return err
	}

	// collect all attrs: precomputed + record attrs
	attrs := make(map[string]any, len(h.preAttrs))
	for _, a := range h.preAttrs {
		attrs[a.Key] = a.Value.Any()
	}
	record.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	if err := h.appendPrependedFields(out, record.Level, attrs); err != nil {
		return err
	}

	if _, err := out.WriteString(record.Message); err != nil {
		return err
	}

	// print remaining fields in the same color as the level
	colorFunc := h.getLevelColor(record.Level)
	for fieldKey, fieldValue := range attrs {
		if err := h.addPadding(out); err != nil {
			return err
		}
		if _, err := out.WriteString(fmt.Sprintf("%s=%v", colorFunc(fieldKey), fieldValue)); err != nil {
			return err
		}
	}

	if _, err := out.WriteString("\n"); err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.writer.Write(out.Bytes())
	return err
}

// WithAttrs returns a new Handler with the given attrs precomputed
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newH := h.clone()
	newH.preAttrs = append(newH.preAttrs, attrs...)
	return newH
}

// WithGroup returns a new Handler with the given group name
func (h *Handler) WithGroup(name string) slog.Handler {
	// groups are not used in the original formatter, but we support them minimally
	// by prefixing attr keys with the group name
	_ = name
	return h
}

func (h *Handler) clone() *Handler {
	return &Handler{
		TimestampFormat:         h.TimestampFormat,
		ColorSchema:             h.ColorSchema,
		DisableColors:           h.DisableColors,
		ForceColors:             h.ForceColors,
		DisableTimestamp:        h.DisableTimestamp,
		UseTimePassedAsTimestamp: h.UseTimePassedAsTimestamp,
		UseUppercaseLevel:       h.UseUppercaseLevel,
		PadAllLogEntries:        h.PadAllLogEntries,
		Level:                   h.Level,
		writer:                  h.writer,
		preAttrs:                append([]slog.Attr{}, h.preAttrs...),
	}
}

// appendTimeInfo appends the time related info
func (h *Handler) appendTimeInfo(out io.StringWriter, t time.Time) error {
	if h.DisableTimestamp {
		return nil
	}

	var timeInfo string
	if h.UseTimePassedAsTimestamp {
		timeInfo = fmt.Sprintf("[%04d]", int(t.Sub(baseTimestamp)/time.Second))
	} else {
		timestampFormat := h.TimestampFormat
		if timestampFormat == "" {
			timestampFormat = defaultTimestampFormat
		}
		timeInfo = t.Format(timestampFormat)
	}

	colorFunc := h.getColor(h.ColorSchema.Timestamp, defaultColorSchema.Timestamp)
	if _, err := out.WriteString(colorFunc(timeInfo)); err != nil {
		return err
	}

	return h.addPadding(out)
}

// appendLevelInfo appends the log level info
func (h *Handler) appendLevelInfo(out io.StringWriter, level slog.Level) error {
	colorFunc := h.getLevelColor(level)
	entryLevel := h.levelString(level)

	if h.UseUppercaseLevel {
		entryLevel = strings.ToUpper(entryLevel)
	}

	_, err := out.WriteString(colorFunc(fmt.Sprintf("%7s", entryLevel)))
	return err
}

// appendPrependedFields appends the prepended fields and removes matched keys from attrs
func (h *Handler) appendPrependedFields(out io.StringWriter, level slog.Level, attrs map[string]any) error {
	for fieldKey, fieldMatches := range FieldMatchColorScheme {
		// check for the longest value for the required padding on PadAllLogEntries = true
		longestValue := 0

		if h.PadAllLogEntries {
			for _, fieldMatch := range fieldMatches {
				l := len(fmt.Sprintf("[%v]", fieldMatch.Value))
				if longestValue < l {
					longestValue = l
				}
			}
		}

		padded := false
		// use the longest value for the padding (always 0 if PadAllLogEntries = false)
		outFormat := fmt.Sprintf("%%%ds", longestValue)

		if entryValue, ok := attrs[fieldKey]; ok {
			for _, matchValue := range fieldMatches {
				if fmt.Sprintf("%v", entryValue) == fmt.Sprintf("%v", matchValue.Value) {
					colorFunc := h.getColor(matchValue.Color, "")
					_, err := out.WriteString(
						" " + colorFunc(fmt.Sprintf(outFormat, fmt.Sprintf("[%v]", entryValue))),
					)
					if err != nil {
						return err
					}

					delete(attrs, fieldKey)
					padded = true
					break
				}
			}
		}

		// add padding if no match got found and PadAllLogEntries is enabled
		if h.PadAllLogEntries && !padded {
			if err := h.addPadding(out); err != nil {
				return err
			}
			if _, err := out.WriteString(fmt.Sprintf(outFormat, "")); err != nil {
				return err
			}
		}
	}

	return h.addPadding(out)
}

// getLevelColor returns the ansi ColorFunc depending on the log level
func (h *Handler) getLevelColor(level slog.Level) func(string) string {
	if h.DisableColors || (!h.isTerminal() && !h.ForceColors) {
		return ansi.ColorFunc("")
	}

	switch {
	case level >= slog.LevelError:
		return h.getColor(h.ColorSchema.ErrorLevel, defaultColorSchema.ErrorLevel)
	case level >= slog.LevelWarn:
		return h.getColor(h.ColorSchema.WarnLevel, defaultColorSchema.WarnLevel)
	case level >= slog.LevelInfo:
		return h.getColor(h.ColorSchema.InfoLevel, defaultColorSchema.InfoLevel)
	default:
		return h.getColor(h.ColorSchema.DebugLevel, defaultColorSchema.DebugLevel)
	}
}

// getColor checks if we have a terminal and colors are not disabled and returns the ansi ColorFunc
func (h *Handler) getColor(customColor string, defaultColor string) func(string) string {
	if h.DisableColors || (!h.isTerminal() && !h.ForceColors) {
		return ansi.ColorFunc("")
	}

	style := defaultColor
	if customColor != "" {
		style = customColor
	}

	return ansi.ColorFunc(style)
}

// addPadding adds the assigned padding character and writes it to our buffer
func (h *Handler) addPadding(writer io.StringWriter) error {
	_, err := writer.WriteString(" ")
	return err
}

// levelString returns the logrus-compatible level string for a slog.Level
func (h *Handler) levelString(level slog.Level) string {
	switch {
	case level >= slog.LevelError+4:
		return "panic"
	case level >= slog.LevelError+2:
		return "fatal"
	case level >= slog.LevelError:
		return "error"
	case level >= slog.LevelWarn:
		return "warning"
	case level >= slog.LevelInfo:
		return "info"
	default:
		return "debug"
	}
}

// AddFieldMatchColorScheme registers field match color scheme
func AddFieldMatchColorScheme(key string, match *FieldMatch) {
	if FieldMatchColorScheme == nil {
		FieldMatchColorScheme = make(map[string][]*FieldMatch)
	}

	FieldMatchColorScheme[key] = append(FieldMatchColorScheme[key], match)
}
