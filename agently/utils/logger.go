package utils

import (
	"log"
	"os"
	"strings"
	"sync"
)

type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARNING"
	LevelError LogLevel = "ERROR"
)

type AgentlyLogger struct {
	mu     sync.RWMutex
	logger *log.Logger
	level  LogLevel
}

func NewLogger(appName string, level LogLevel) *AgentlyLogger {
	if appName == "" {
		appName = "Agently"
	}
	if level == "" {
		level = LevelInfo
	}
	return &AgentlyLogger{
		logger: log.New(os.Stdout, "["+appName+"] ", log.LstdFlags),
		level:  level,
	}
}

func (l *AgentlyLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *AgentlyLogger) Level() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

func (l *AgentlyLogger) enabled(level LogLevel) bool {
	levels := map[LogLevel]int{LevelDebug: 1, LevelInfo: 2, LevelWarn: 3, LevelError: 4}
	cur := levels[l.Level()]
	want := levels[level]
	if want == 0 {
		want = 2
	}
	if cur == 0 {
		cur = 2
	}
	return want >= cur
}

func (l *AgentlyLogger) log(level LogLevel, msg string) {
	if !l.enabled(level) {
		return
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	prefix := "[" + strings.ToUpper(string(level)) + "] "
	l.logger.Println(prefix + msg)
}

func (l *AgentlyLogger) Debug(msg string) { l.log(LevelDebug, msg) }
func (l *AgentlyLogger) Info(msg string)  { l.log(LevelInfo, msg) }
func (l *AgentlyLogger) Warn(msg string)  { l.log(LevelWarn, msg) }
func (l *AgentlyLogger) Error(msg string) { l.log(LevelError, msg) }
