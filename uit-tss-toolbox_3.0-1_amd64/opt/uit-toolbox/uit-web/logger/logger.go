package logger

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
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
	HTTPAuth(req *http.Request, message string)
	HTTPDebug(req *http.Request, message string)
	HTTPInfo(req *http.Request, message string)
	HTTPWarning(req *http.Request, message string)
	HTTPError(req *http.Request, message string)
}

type ConsoleLogger struct {
	Level   atomic.Int64
	writeMu sync.Mutex
}

func (consoleLogger *ConsoleLogger) SetLoggerLevel(logLevel LogLevel) {
	consoleLogger.Level.Store(int64(logLevel))
}

func (consoleLogger *ConsoleLogger) logHTTPErr(req *http.Request, logLevel LogLevel, message string) {
	if int64(logLevel) < consoleLogger.Level.Load() {
		return
	}

	// Buffer values outside of lock
	ipAddr, _, _ := net.SplitHostPort(req.RemoteAddr)
	requestMethod := req.Method
	requestURI := req.URL.Path
	requestInfo := fmt.Sprintf(" (%s %s %s)", ipAddr, requestMethod, requestURI)
	consoleLogger.log(logLevel, message+requestInfo)
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
	formattedMessage := fmt.Sprintf("%s [%s] %s\n", currentTime, logLevel.getLogLevel(), message)
	writer := bufio.NewWriter(output)
	consoleLogger.writeMu.Lock()
	writer.Write([]byte(formattedMessage))
	writer.Flush()
	consoleLogger.writeMu.Unlock()
}

func (consoleLogger *ConsoleLogger) Auth(message string)    { consoleLogger.log(Auth, message) }
func (consoleLogger *ConsoleLogger) Debug(message string)   { consoleLogger.log(Debug, message) }
func (consoleLogger *ConsoleLogger) Info(message string)    { consoleLogger.log(Info, message) }
func (consoleLogger *ConsoleLogger) Warning(message string) { consoleLogger.log(Warning, message) }
func (consoleLogger *ConsoleLogger) Error(message string)   { consoleLogger.log(Error, message) }

func (consoleLogger *ConsoleLogger) HTTPAuth(req *http.Request, message string) {
	consoleLogger.logHTTPErr(req, Auth, message)
}
func (consoleLogger *ConsoleLogger) HTTPDebug(req *http.Request, message string) {
	consoleLogger.logHTTPErr(req, Debug, message)
}
func (consoleLogger *ConsoleLogger) HTTPInfo(req *http.Request, message string) {
	consoleLogger.logHTTPErr(req, Info, message)
}
func (consoleLogger *ConsoleLogger) HTTPWarning(req *http.Request, message string) {
	consoleLogger.logHTTPErr(req, Warning, message)
}
func (consoleLogger *ConsoleLogger) HTTPError(req *http.Request, message string) {
	consoleLogger.logHTTPErr(req, Error, message)
}

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
