# Colored Nested Slog Handler
[![tests](https://github.com/DaRealFreak/colored-nested-formatter/actions/workflows/tests.yml/badge.svg)](https://github.com/DaRealFreak/colored-nested-formatter/actions/workflows/tests.yml) [![build](https://github.com/DaRealFreak/colored-nested-formatter/actions/workflows/build.yml/badge.svg)](https://github.com/DaRealFreak/colored-nested-formatter/actions/workflows/build.yml) [![golangci-lint](https://github.com/DaRealFreak/colored-nested-formatter/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/DaRealFreak/colored-nested-formatter/actions/workflows/golangci-lint.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/DaRealFreak/colored-nested-formatter/v2)](https://goreportcard.com/report/github.com/DaRealFreak/colored-nested-formatter/v2) ![License](https://img.shields.io/github/license/DaRealFreak/colored-nested-formatter)

Human readable log handler for the standard library `log/slog` package. Option to define custom colors for specific field matches. Drop-in successor to [colored-nested-formatter v1](https://github.com/DaRealFreak/colored-nested-formatter/tree/v1.0.1) (logrus).

## Installation
```bash
go get github.com/DaRealFreak/colored-nested-formatter/v2
```

## Usage
```go
package main

import (
	"log/slog"
	"time"

	formatter "github.com/DaRealFreak/colored-nested-formatter/v2"
	"github.com/mattn/go-colorable"
)

func main() {
	handler := formatter.NewHandler(colorable.NewColorableStdout(), &formatter.Handler{
		DisableColors:           false,
		ForceColors:             false,
		DisableTimestamp:        false,
		UseUppercaseLevel:       false,
		UseTimePassedAsTimestamp: false,
		TimestampFormat:         time.StampMilli,
		PadAllLogEntries:        true,
		Level:                   slog.LevelDebug,
	})

	logger := slog.New(handler)

	formatter.AddFieldMatchColorScheme("color", &formatter.FieldMatch{
		Value: "blue",
		Color: "232:33",
	})
	formatter.AddFieldMatchColorScheme("moreColorFields", &formatter.FieldMatch{
		Value: "green",
		Color: "232:34",
	})

	logger.Info("normal info log entry")
	logger.Info("normal colored not nested info log entry", "color", "unregistered")
	logger.Info("blue colored nested info log entry", "color", "blue")
	logger.Info("blue and green colored nested info log entry", "color", "blue", "moreColorFields", "green")
}
```

### Using WithAttrs (replaces logrus WithField)
```go
// create a sub-logger with a precomputed field (like logrus.WithField)
moduleLogger := slog.New(handler).With("module", "deviantart.com")
moduleLogger.Info("downloading updates")
moduleLogger.Warn("rate limit approaching")
```

### Setting as default logger
```go
slog.SetDefault(slog.New(handler))

// then use the global functions
slog.Info("message", "module", "pixiv.net")
```

## Migration from v1 (logrus)

| v1 (logrus) | v2 (slog) |
|---|---|
| `logrus.WithField("k", v).Info("msg")` | `slog.Info("msg", "k", v)` |
| `logrus.WithFields(logrus.Fields{...}).Info("msg")` | `slog.Info("msg", "k1", v1, "k2", v2)` |
| `log.SetFormatter(&formatter.Formatter{...})` | `slog.SetDefault(slog.New(formatter.NewHandler(w, &formatter.Handler{...})))` |
| `formatter.AddFieldMatchColorScheme(...)` | `formatter.AddFieldMatchColorScheme(...)` (unchanged) |

## Configuration
```go
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
	// minimum log level (default: slog.LevelDebug)
	Level slog.Level
}
```

### Custom log levels
`slog` doesn't have `Fatal` and `Panic` levels natively. This handler maps them to custom levels for color support:

| Level | slog value | Color |
|---|---|---|
| Debug | `slog.LevelDebug` | blue |
| Info | `slog.LevelInfo` | green |
| Warn | `slog.LevelWarn` | yellow |
| Error | `slog.LevelError` | red |
| Fatal | `slog.LevelError + 2` | red |
| Panic | `slog.LevelError + 4` | red |

## Development
Want to contribute? Great!
I'm always glad hearing about bugs or pull requests.

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details
