package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
)

type Level string

const (
	INFO  Level = "INFO"
	WARN  Level = "WARN"
	ERROR Level = "ERROR"
)

func (l Level) AnsiString() string {
	switch l {
	case WARN:
		return internal.ColorYellow + "WARN" + internal.AnsiReset
	case ERROR:
		return internal.ColorRed + "ERROR" + internal.AnsiReset
	default:
		return internal.ColorGreen + "INFO" + internal.AnsiReset
	}
}

type Logger struct {
	logger *log.Logger
	source string
	mu     sync.Mutex
}

var (
	defaultLogger = New("Server")
)

// New creates a new logger with the given source name
func New(source string) *Logger {
	return &Logger{
		source: source,
		logger: log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) output(level Level, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timeStr := time.Now().Format("15:04:05")
	prefix := fmt.Sprintf("[%s] [%s/%s]: %s", timeStr, l.source, level.AnsiString(), msg)
	l.logger.Println(prefix)
}

func (l *Logger) Info(format string, v ...any) {
	l.output(INFO, fmt.Sprintf(format, v...))
}

func (l *Logger) Warn(format string, v ...any) {
	l.output(WARN, fmt.Sprintf(format, v...))
}

func (l *Logger) Error(format string, v ...any) {
	l.output(ERROR, fmt.Sprintf(format, v...))
}

func (l *Logger) Component(level Level, c tc.Component) {
	for _, line := range c.AnsiLines() {
		l.output(level, line)
	}
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
	defaultLogger.Error(format, v...)
	os.Exit(1)
}

func Component(level Level, c tc.Component) {
	defaultLogger.Component(level, c)
}
