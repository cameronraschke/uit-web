package logger

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type LogLevel int64

const (
	Auth LogLevel = iota
	Debug
	Info
	Warning
	Error
)

// WIP
type loggerType int64

const (
	Console loggerType = iota
	File
)

func (logLevel LogLevel) getLogLevel() string {
	switch logLevel {
	case Auth:
		return "AUTH"
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warning:
		return "WARNING"
	case Error:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "auth":
		return Auth
	case "debug":
		return Debug
	case "info":
		return Info
	case "warning", "warn":
		return Warning
	case "error":
		return Error
	default:
		return Info
	}
}

func TimePrefix() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

type Logger interface {
	SetLoggerLevel(logLevel LogLevel)
	Auth(message string)
	Debug(message string)
	Info(message string)
	Warning(message string)
	Error(message string)
}

type ConsoleLogger struct {
	Level   atomic.Int64
	writeMu sync.Mutex
}

func (consoleLogger *ConsoleLogger) SetLoggerLevel(logLevel LogLevel) {
	consoleLogger.Level.Store(int64(logLevel))
}

func (consoleLogger *ConsoleLogger) log(logLevel LogLevel, message string) {
	if int64(logLevel) < consoleLogger.Level.Load() {
		return
	}
	output := os.Stdout
	if logLevel >= Warning {
		output = os.Stderr
	}

	// Buffer values outside of lock
	currentTime := TimePrefix()
	currentLogLevel := logLevel.getLogLevel()
	formattedMessage := fmt.Sprintf("%s [%s] %s\n", currentTime, currentLogLevel, message)
	consoleLogger.writeMu.Lock()
	output.Write([]byte(formattedMessage))
	consoleLogger.writeMu.Unlock()
}

func (consoleLogger *ConsoleLogger) Auth(message string)    { consoleLogger.log(Auth, message) }
func (consoleLogger *ConsoleLogger) Debug(message string)   { consoleLogger.log(Debug, message) }
func (consoleLogger *ConsoleLogger) Info(message string)    { consoleLogger.log(Info, message) }
func (consoleLogger *ConsoleLogger) Warning(message string) { consoleLogger.log(Warning, message) }
func (consoleLogger *ConsoleLogger) Error(message string)   { consoleLogger.log(Error, message) }

func CreateLogger(loggerType string, logLevel LogLevel) Logger {
	switch strings.ToLower(loggerType) {
	case "console":
		logger := &ConsoleLogger{}
		logger.SetLoggerLevel(logLevel)
		return logger
	default:
		logger := &ConsoleLogger{}
		logger.SetLoggerLevel(Warning)
		return logger
	}
}
