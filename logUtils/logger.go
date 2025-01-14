package logUtils

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
)

type Level int

const (
	_                = iota
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

type Logger struct {
	loggers map[Level]*log.Logger
	level   Level
}

var (
	lOnce          sync.Once
	loggerInstance *Logger
)

func NewLogger(level string) (*Logger, error) {
	var l Level

	switch level {
	case "debug", "DEBUG":
		l = DebugLevel
	case "info", "INFO":
		l = InfoLevel
	case "warn", "WARN":
		l = WarnLevel
	case "error", "ERROR":
		l = ErrorLevel
	default:
		return nil, errors.New("invalid log level")
	}

	lOnce.Do(func() {
		loggerInstance = &Logger{
			loggers: map[Level]*log.Logger{
				DebugLevel: log.New(os.Stdout, "\033[0m[DEBUG] ", log.LstdFlags),
				InfoLevel:  log.New(os.Stdout, "\033[0m[INFO] ", log.LstdFlags),
				WarnLevel:  log.New(os.Stdout, "\033[31m[WARN] ", log.LstdFlags),
				ErrorLevel: log.New(os.Stderr, "\033[31m[ERROR] \033[0m", log.LstdFlags),
			},
			level: l,
		}
	})

	return loggerInstance, nil
}

func (l *Logger) Log(level Level, v ...interface{}) {
	if l.level <= level {
		l.loggers[level].Println(v...)
	}
}

func (l *Logger) Logf(level Level, format string, v ...interface{}) {
	if l.level <= level {
		l.loggers[level].Printf(format, v...)
	}
}

func (l *Logger) Debug(v ...interface{}) {
	l.Log(DebugLevel, v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.Logf(DebugLevel, format, v...)
}

func (l *Logger) Info(v ...interface{}) {
	l.Log(InfoLevel, v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.Logf(InfoLevel, format, v...)
}

func (l *Logger) Warn(v ...interface{}) {
	l.Log(WarnLevel, v...)
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	l.Logf(WarnLevel, format, v...)
}

func (l *Logger) Error(v ...interface{}) {
	if l.level <= ErrorLevel {
		_, file, line, ok := runtime.Caller(1) // Adjust the depth based on your structure
		if ok {
			message := fmt.Sprintf("%s:%d: %s", file, line, fmt.Sprint(v...))
			l.loggers[ErrorLevel].Println(message)
		} else {
			l.loggers[ErrorLevel].Println(fmt.Sprintln(v...))
		}
	}
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.level <= ErrorLevel {
		_, file, line, ok := runtime.Caller(1) // Adjust the depth based on your structure
		if ok {
			message := fmt.Sprintf(format, v...)
			l.loggers[ErrorLevel].Printf("%s:%d: %s", file, line, message)
		} else {
			l.loggers[ErrorLevel].Printf("Error: %s", format)
		}
	}
}
