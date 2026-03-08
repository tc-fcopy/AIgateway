package observability

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	GlobalLogger = NewLogger()
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Entry struct {
	logger *Logger
	fields map[string]interface{}
}

type Logger struct {
	mu         sync.RWMutex
	level      LogLevel
	jsonOutput bool
	requestID  string
	baseFields map[string]interface{}
	out        *log.Logger
}

func NewLogger() *Logger {
	return &Logger{
		level:      LevelInfo,
		jsonOutput: strings.EqualFold(os.Getenv("LOG_JSON"), "true"),
		baseFields: map[string]interface{}{},
		out:        log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) SetRequestID(requestID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.requestID = requestID
}

func (l *Logger) ClearRequestID() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.requestID = ""
}

func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) WithFields(fields map[string]interface{}) *Entry {
	copied := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		copied[k] = v
	}
	return &Entry{logger: l, fields: copied}
}

func (e *Entry) Info(format string, args ...interface{}) {
	e.logger.log(LevelInfo, e.fields, format, args...)
}

func (e *Entry) Warn(format string, args ...interface{}) {
	e.logger.log(LevelWarn, e.fields, format, args...)
}

func (e *Entry) Error(format string, args ...interface{}) {
	e.logger.log(LevelError, e.fields, format, args...)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, nil, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, nil, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, nil, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, nil, format, args...)
}

func (l *Logger) log(level LogLevel, fields map[string]interface{}, format string, args ...interface{}) {
	l.mu.RLock()
	current := l.level
	requestID := l.requestID
	jsonOutput := l.jsonOutput
	l.mu.RUnlock()

	if level < current {
		return
	}

	message := format
	if len(args) > 0 {
		message = fmt.Sprintf(format, args...)
	}

	payload := map[string]interface{}{
		"time":    time.Now().Format(time.RFC3339Nano),
		"level":   levelString(level),
		"message": message,
	}
	if requestID != "" {
		payload["request_id"] = requestID
	}
	for k, v := range fields {
		payload[k] = v
	}

	if jsonOutput {
		b, _ := json.Marshal(payload)
		l.out.Println(string(b))
		return
	}

	l.out.Printf("[%s] %s", levelString(level), message)
}

func levelString(level LogLevel) string {
	switch level {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	default:
		return "error"
	}
}
