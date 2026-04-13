package logger

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	CRITICAL
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case CRITICAL:
		return "CRITICAL"
	default:
		return "INFO"
	}
}

func (l Level) AnsiString() string {
	switch l {
	case DEBUG:
		return internal.ColorLightBlue + "DEBUG" + internal.AnsiReset
	case WARN:
		return internal.ColorYellow + "WARN" + internal.AnsiReset
	case ERROR:
		return internal.ColorRed + "ERROR" + internal.AnsiReset
	case CRITICAL:
		return internal.ColorRed + internal.AnsiBold + "CRITICAL" + internal.AnsiReset
	default:
		return internal.ColorGreen + "INFO" + internal.AnsiReset
	}
}

func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return DEBUG
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	case "critical":
		return CRITICAL
	default:
		return INFO
	}
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type fileWriter struct {
	w io.Writer
}

func (fw *fileWriter) Write(p []byte) (int, error) {
	cleaned := ansiRegex.ReplaceAll(p, nil)
	return fw.w.Write(cleaned)
}

type Logger struct {
	source   string
	mu       sync.Mutex
	console  io.Writer
	file     io.Writer
	minLevel Level
}

var defaultLogger = &Logger{
	source:   "Server",
	console:  os.Stdout,
	minLevel: INFO,
}

func New(source string) *Logger {
	return &Logger{
		source:   source,
		console:  defaultLogger.console,
		file:     defaultLogger.file,
		minLevel: defaultLogger.minLevel,
	}
}

func Configure(level string, filePath string) error {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()

	defaultLogger.minLevel = ParseLevel(level)

	if filePath != "" {
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		defaultLogger.file = &fileWriter{w: f}
	}

	return nil
}

func (l *Logger) output(level Level, msg string) {
	if level < l.minLevel {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timeStr := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] [%s/%s]: %s\n", timeStr, l.source, level.AnsiString(), msg)
	_, _ = l.console.Write([]byte(line))

	if l.file != nil {
		plainLine := fmt.Sprintf("[%s] [%s/%s]: %s\n", timeStr, l.source, level.String(), msg)
		_, _ = l.file.Write([]byte(plainLine))
	}
}

func (l *Logger) Debug(format string, v ...any) {
	if l.minLevel > DEBUG {
		return
	}
	l.output(DEBUG, internal.ColorLightBlue+fmt.Sprintf(format, v...)+internal.AnsiReset)
}

func (l *Logger) Info(format string, v ...any) {
	if l.minLevel > INFO {
		return
	}
	l.output(INFO, fmt.Sprintf(format, v...))
}

func (l *Logger) Warn(format string, v ...any) {
	if l.minLevel > WARN {
		return
	}
	l.output(WARN, fmt.Sprintf(format, v...))
}

func (l *Logger) Error(format string, v ...any) {
	l.output(ERROR, fmt.Sprintf(format, v...))
}

func (l *Logger) Component(level Level, c tc.Component) {
	if level < l.minLevel {
		return
	}
	for _, line := range c.AnsiLines() {
		l.output(level, line)
	}
}

func IsDebug() bool {
	return defaultLogger.minLevel <= DEBUG
}

func Debug(format string, v ...any) {
	defaultLogger.Debug(format, v...)
}

func Info(format string, v ...any) {
	defaultLogger.Info(format, v...)
}

func Warn(format string, v ...any) {
	defaultLogger.Warn(format, v...)
}

func Error(format string, v ...any) {
	defaultLogger.Error(format, v...)
}

func Fatal(format string, v ...any) {
	defaultLogger.output(CRITICAL, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func Component(level Level, c tc.Component) {
	defaultLogger.Component(level, c)
}
