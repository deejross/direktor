package logger

import (
	"log"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var levels = map[string]zapcore.Level{
	"DEBUG": zap.DebugLevel,
	"INFO":  zap.InfoLevel,
	"WARN":  zap.WarnLevel,
	"ERROR": zap.ErrorLevel,
	"FATAL": zap.FatalLevel,
}

// New returns a new scoped logger with defaults that formats output based on the `LOG_OUTPUT`
// environment variable. Options include `console` (default) and `json`.
func New(pkg string, fields ...zapcore.Field) *zap.Logger {
	output := strings.ToLower(os.Getenv("LOG_OUTPUT"))
	if output == "json" {
		return NewJSON(pkg, fields...)
	}

	return NewConsole(pkg, fields...)
}

// NewJSON returns a new scoped logger with defaults that outputs to JSON format.
func NewJSON(pkg string, fields ...zapcore.Field) *zap.Logger {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	return get(config, pkg, fields...)
}

// NewConsole returns a new scoped logger with defaults for human-readable output to a console.
func NewConsole(pkg string, fields ...zapcore.Field) *zap.Logger {
	config := zap.NewDevelopmentConfig()
	return get(config, pkg, fields...)
}

// ParseLevel returns the log level fromthe given string, falling back to the given default if
// level string is unknown or empty.
func ParseLevel(level string, defaultLevel zapcore.Level) zapcore.Level {
	level = strings.ToUpper(strings.TrimSpace(level))
	if lev, ok := levels[level]; ok {
		return lev
	}
	return defaultLevel
}

// RedirectLogPackage redirects any output from the built-in `log` package to the given logger.
func RedirectLogPackage(l *zap.Logger) {
	w := &writer{
		l: l,
	}
	log.SetOutput(w)
}

func get(config zap.Config, pkg string, fields ...zapcore.Field) *zap.Logger {
	if fields == nil {
		fields = []zapcore.Field{}
	}

	newFields := make([]zapcore.Field, len(fields)+1)
	newFields[0] = zap.String("pkg", pkg)
	for i, f := range fields {
		newFields[i+1] = f
	}

	config.Level = zap.NewAtomicLevelAt(ParseLevel(os.Getenv("LOG_LEVEL"), zap.InfoLevel))
	l, _ := config.Build(zap.AddStacktrace(zap.ErrorLevel), zap.AddCaller())
	l = l.With(newFields...)
	return l
}

type writer struct {
	l *zap.Logger
}

// Write implements io.Writer interface.
func (w *writer) Write(b []byte) (int, error) {
	w.l.Error(string(b))
	return len(b), nil
}
