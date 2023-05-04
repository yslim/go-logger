// Package logger
// Created by yslim on 2018. 6. 16.
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type LogLevel int
type RollType int

const (
	ALL LogLevel = iota
	TRACE
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
	OFF
)

const (
	RollDaily RollType = 1 << iota
	RollSize
)

var (
	LogLevelName = []string{
		"ALL",
		"TRACE",
		"DEBUG",
		"INFO",
		"WARN",
		"ERROR",
		"FATAL",
		"OFF",
	}

	// 31:red, 32:green, 33:yellow, 34:blue, 35:magenta, 36:cyan, 37:gray, 0:reset
	ColoredLogLevelName = []string{
		"ALL",
		"\033[37mTRACE\033[0m",
		"\033[00mDEBUG\033[0m",
		"\033[32mINFO\033[0m",
		"\033[33mWARN\033[0m",
		"\033[31mERROR\033[0m",
		"\033[31mFATAL\033[0m",
		"OFF",
	}

	singletonInstance *Logger  = nil
	globalLogLevel    LogLevel = ALL
	mutex             sync.Mutex
)

func IsEnabled(lvl LogLevel) bool {
	return lvl >= globalLogLevel
}

func SetLevel(lvl LogLevel) {
	globalLogLevel = lvl
}

/*
 * ILogTarget
 */
type iLogTarget interface {
	Append(msg string)
}

/*
 * logTarget
 */
type logTarget struct {
	sync.Mutex
}

func newTarget() *logTarget {
	return &logTarget{}
}

func (l *logTarget) Append(msg string) {
	fmt.Printf("Append() function not implemented!!!\n")
}

type LogTargetConsole struct {
	logTarget
}

func NewConsole() *LogTargetConsole {
	return &LogTargetConsole{logTarget{}}
}

func (l *LogTargetConsole) Append(msg string) {
	l.Lock()
	defer l.Unlock()
	fmt.Print(msg)
}

type LogTargetFileBySize struct {
	logTarget
	limitSize int64
	numFiles  int
	logPath   string
}

func NewLogTargetFileBySize(limitSize int64, numFiles int, logPath string) *LogTargetFileBySize {
	return &LogTargetFileBySize{logTarget{}, limitSize, numFiles, logPath}
}

func (l *LogTargetFileBySize) Append(msg string) {
	l.Lock()
	defer l.Unlock()

	if fileSize(l.logPath)+int64(len(msg)) > l.limitSize {
		l.RotateLogFiles()
	}

	f, err := os.OpenFile(l.logPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Printf("[ logTargetFileBySize:Append() ] file(\"%s\") open failed, error=%v",
			l.logPath, err.Error())
		return
	}
	_, _ = f.WriteString(msg)
	_ = f.Close()
}

func (l *LogTargetFileBySize) RotateLogFiles() {
	for i := l.numFiles - 2; i > 0; i-- {
		_ = os.Rename(fmt.Sprint(l.logPath, ".", i), fmt.Sprint(l.logPath, ".", i+1))
	}
	_ = os.Rename(l.logPath, fmt.Sprint(l.logPath, ".", 1))
}

type LogTargetFileDaily struct {
	logTarget
	logPath string
}

func NewLogTargetFileDaily(logPath string) *LogTargetFileDaily {
	return &LogTargetFileDaily{logTarget{}, logPath}
}

func (l *LogTargetFileDaily) Append(msg string) {
	l.Lock()
	defer l.Unlock()

	now := time.Now()
	logDir := fmt.Sprintf("%s/%02d", l.logPath, int(now.Month()))

	if !isExist(logDir) {
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			fmt.Printf("[ logTargetFileDaily:Append() ] Mkdir(\"%s\") failed, error=%v", logDir, err.Error())
			return
		}
	}

	logPath := fmt.Sprintf("%s/%02d%02d.log", logDir, int(now.Month()), now.Day())

	// if over one year then remove it
	if now.Sub(fileLastModifiedTime(logPath)).Hours() > 24*360 {
		_ = os.Remove(logPath)
	}

	// although Open/Close for each log decreases performance, but
	// I want to save to disk when append is performed
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Printf("[ logTargetFileDaily:Append() ] file(\"%s\") open failed, error=%v",
			logPath, err.Error())
		os.Exit(1)
	}
	_, _ = f.WriteString(msg)
	_ = f.Close()
}

type Logger struct {
	logTargets             []iLogTarget
	isReady                bool
	useColoredLogLevelName bool
	callDepth              int
}

func NewLogger(useColoredLogLevelName bool) *Logger {
	return &Logger{[]iLogTarget{}, false, useColoredLogLevelName, 2}
}

func (l *Logger) AddTarget(target iLogTarget) {
	l.logTargets = append(l.logTargets, target)
	l.isReady = true
}

func (l *Logger) SetCallDepth(callDepth int) {
	l.callDepth = callDepth
}

func (l *Logger) GetCallDepth() int {
	return l.callDepth
}

func (l *Logger) Print(v ...interface{}) {
	fmt.Print(v...)
}

func (l *Logger) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func (l *Logger) Trace(v ...interface{}) {
	l.logFormat(TRACE, l.callDepth, fmt.Sprint(v...))
}

func (l *Logger) Tracef(format string, v ...interface{}) {
	l.logFormat(TRACE, l.callDepth, fmt.Sprintf(format, v...))
}

func (l *Logger) Debug(v ...interface{}) {
	l.logFormat(DEBUG, l.callDepth, fmt.Sprint(v...))
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.logFormat(DEBUG, l.callDepth, fmt.Sprintf(format, v...))
}

func (l *Logger) Info(v ...interface{}) {
	l.logFormat(INFO, l.callDepth, fmt.Sprint(v...))
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.logFormat(INFO, l.callDepth, fmt.Sprintf(format, v...))
}

func (l *Logger) Warn(v ...interface{}) {
	l.logFormat(WARN, l.callDepth, fmt.Sprint(v...))
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	l.logFormat(WARN, l.callDepth, fmt.Sprintf(format, v...))
}

func (l *Logger) Error(v ...interface{}) {
	l.logFormat(ERROR, l.callDepth, fmt.Sprint(v...))
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.logFormat(ERROR, l.callDepth, fmt.Sprintf(format, v...))
}

func (l *Logger) Fatal(v ...interface{}) {
	l.logFormat(FATAL, l.callDepth, fmt.Sprint(v...))
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logFormat(FATAL, l.callDepth, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *Logger) logFormat(lvl LogLevel, callDepth int, msg string) {
	var sb strings.Builder
	now := time.Now()

	year, month, day := now.Date()
	// Time and LogLevel
	if l.useColoredLogLevelName {
		sb.WriteString(fmt.Sprintf("[%04d-%02d-%02d %02d:%02d:%02d] %-14s ",
			year, int(month), day, now.Hour(), now.Minute(), now.Second(), ColoredLogLevelName[lvl]))
	} else {
		sb.WriteString(fmt.Sprintf("[%04d-%02d-%02d %02d:%02d:%02d] %-5s ",
			year, int(month), day, now.Hour(), now.Minute(), now.Second(), LogLevelName[lvl]))
	}
	// Mesg
	sb.WriteString(msg)
	// File & Line
	_, file, line, ok := runtime.Caller(callDepth)
	if !ok {
		file = "???"
		line = 0
	}
	sb.WriteString(fmt.Sprintf(" [%s:%d]\n", filepath.Base(file), line))

	l.log(lvl, sb.String())
}

func (l *Logger) log(lvl LogLevel, msg string) {
	if !l.isReady {
		fmt.Println("[ Logger ] log path is not set.")
	}

	if !IsEnabled(lvl) {
		return
	}

	for _, v := range l.logTargets {
		v.Append(msg)
	}
}

// GetLogger : Logger Factory
func GetLogger() *Logger {
	if singletonInstance == nil {
		fmt.Println("[ GetLoggerInstance ] Logger is not created, use InitLogger...")
		os.Exit(1)
	}
	return singletonInstance
}

// InitLogger
//
// lvl : log level
// limitSize : max log file size limit to rotate, only for RollSize
// numFiles: number of log files to be maintained for rotating, only for RollSize
// logPath : log path + name (/a/b/c/message)
// rollType : log file rolling by Daily or Size
// useColoredLogLevelName : colored log level name or not
// force : force re-create singletonInstance when already exists
func InitLogger(lvl LogLevel, limitSize int64, numFiles int, logPath string,
	rollType RollType, useColoredLogLevelName bool, force bool) *Logger {
	mutex.Lock()
	defer mutex.Unlock()

	if singletonInstance != nil && !force {
		return singletonInstance
	}

	singletonInstance = NewLogger(useColoredLogLevelName)

	// set log level
	SetLevel(lvl)

	// add Console
	singletonInstance.AddTarget(NewConsole())

	// add FileLog
	if rollType == RollDaily {
		singletonInstance.AddTarget(NewLogTargetFileDaily(logPath))
	} else {
		singletonInstance.AddTarget(NewLogTargetFileBySize(limitSize, numFiles, logPath))
	}

	singletonInstance.isReady = true

	return singletonInstance
}

func GetLevelByName(logName string) LogLevel {
	for i, v := range LogLevelName {
		if strings.EqualFold(v, logName) {
			return LogLevel(i)
		}
	}
	return 0
}

// ------------------------------------------------------------------------------

func fileSize(path string) int64 {
	f, err := os.Stat(path)
	if err != nil {
		// fmt.Println(err.Error())
		return 0
	}
	return f.Size()
}

func fileLastModifiedTime(path string) time.Time {
	f, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Now()
		} else {
			return time.Time{}
		}
	}
	return f.ModTime()
}

func isExist(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
