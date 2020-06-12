package log

import (
	"fmt"
	"syscall/js"
)

var (
	console  = js.Global().Get("console")
	logLevel = consoleLog
)

type consoleType int

const (
	consoleDebug consoleType = iota
	consoleLog
	consoleWarn
	consoleError
)

func (c consoleType) String() string {
	switch c {
	case consoleDebug:
		return "debug"
	case consoleWarn:
		return "warn"
	case consoleError:
		return "error"
	default:
		return "log"
	}
}

func Debugf(format string, args ...interface{}) int {
	return logf(consoleDebug, format, args...)
}

func Printf(format string, args ...interface{}) int {
	return logf(consoleLog, format, args...)
}

func Warnf(format string, args ...interface{}) int {
	return logf(consoleWarn, format, args...)
}

func Errorf(format string, args ...interface{}) int {
	return logf(consoleError, format, args...)
}

func logf(kind consoleType, format string, args ...interface{}) int {
	if kind < logLevel {
		return 0
	}
	s := fmt.Sprintf(format, args...)
	console.Call(kind.String(), s)
	return len(s)
}

func Debug(args ...interface{}) int {
	return log(consoleDebug, args...)
}

func Print(args ...interface{}) int {
	return log(consoleLog, args...)
}

func Warn(args ...interface{}) int {
	return log(consoleWarn, args...)
}

func Error(args ...interface{}) int {
	return log(consoleError, args...)
}

func log(kind consoleType, args ...interface{}) int {
	if kind < logLevel {
		return 0
	}
	s := fmt.Sprint(args...)
	console.Call(kind.String(), s)
	return len(s)
}

func DebugJSValues(args ...js.Value) int {
	return logJSValues(consoleDebug, args...)
}

func PrintJSValues(args ...js.Value) int {
	return logJSValues(consoleLog, args...)
}

func WarnJSValues(args ...js.Value) int {
	return logJSValues(consoleWarn, args...)
}

func ErrorJSValues(args ...js.Value) int {
	return logJSValues(consoleError, args...)
}

func logJSValues(kind consoleType, args ...js.Value) int {
	if kind < logLevel {
		return 0
	}
	var intArgs []interface{}
	for _, arg := range args {
		intArgs = append(intArgs, arg)
	}
	console.Call(kind.String(), intArgs...)
	return 0
}