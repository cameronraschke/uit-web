package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type LogLevel int32

const (
	Auth LogLevel = iota
	Debug
	Info
	Warning
	Error
)

// WIP
type loggerType int32

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
		return Info // or your default
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
	Level   atomic.Int32
	writeMu sync.Mutex
}

func (consoleLogger *ConsoleLogger) SetLoggerLevel(logLevel LogLevel) {
	consoleLogger.Level.Store(int32(logLevel))
}

func (consoleLogger *ConsoleLogger) log(logLevel LogLevel, message string) {
	if int32(logLevel) >= consoleLogger.Level.Load() {
		var output io.Writer = os.Stdout
		if logLevel >= Warning {
			output = os.Stderr
		} else if logLevel < Warning {
			output = os.Stdout
		} else {
			fmt.Fprintf(os.Stderr, "Unknown log level: %s\n", logLevel.getLogLevel())
			return
		}
		consoleLogger.writeMu.Lock()
		fmt.Fprintf(output, "%s [%s] %s\n", TimePrefix(), logLevel.getLogLevel(), message)
		consoleLogger.writeMu.Unlock()
	}
}

func (consoleLogger *ConsoleLogger) Auth(message string)    { consoleLogger.log(Auth, message) }
func (consoleLogger *ConsoleLogger) Debug(message string)   { consoleLogger.log(Debug, message) }
func (consoleLogger *ConsoleLogger) Info(message string)    { consoleLogger.log(Info, message) }
func (consoleLogger *ConsoleLogger) Warning(message string) { consoleLogger.log(Warning, message) }
func (consoleLogger *ConsoleLogger) Error(message string)   { consoleLogger.log(Error, message) }

func CreateLogger(loggerType string, logLevel LogLevel) Logger {
	switch strings.ToLower(loggerType) {
	case "console":
		cl := &ConsoleLogger{}
		cl.SetLoggerLevel(logLevel)
		return cl
	default:
		cl := &ConsoleLogger{}
		cl.SetLoggerLevel(Warning)
		return cl
	}
}
