//
// Created by yslim on 2018. 6. 16.
//
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
   ALL   LogLevel = iota
   TRACE
   DEBUG
   INFO
   WARN
   ERROR
   FATAL
   OFF
)

const callDepth = 1

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

   singletonInstance *Logger = nil
   mutex             sync.Mutex
)

/*
 * ILogTarget
 */
type iLogTarget interface {
   IsEnabled(level LogLevel) bool
   Append(msg string)
}

/*
 * logTarget
 */
type logTarget struct {
   sync.Mutex
   Level LogLevel
}

func newTarget(lvl LogLevel) *logTarget {
   return &logTarget{Level: lvl}
}

func (l *logTarget) IsEnabled(level LogLevel) bool {
   return level >= l.Level
}

func (l *logTarget) Append(msg string) {
}

/*
 * logTargetConsole
 */
type logTargetConsole struct {
   logTarget
}

func NewConsole(lvl LogLevel) *logTargetConsole {
   return &logTargetConsole{logTarget{Level: lvl}}
}

func (l *logTargetConsole) Append(msg string) {
   l.Lock()
   defer l.Unlock()
   fmt.Print(msg)
}

/*
 * logTargetFileBySize
 */
type logTargetFileBySize struct {
   logTarget
   limitSize int64
   numFiles  int
   logPath   string
}

func NewLogTargetFileBySize(lvl LogLevel, limitSize int64, numFiles int, logPath string) *logTargetFileBySize {
   return &logTargetFileBySize{logTarget{Level: lvl}, limitSize, numFiles, logPath}
}

func (l *logTargetFileBySize) Append(msg string) {
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

func (l *logTargetFileBySize) RotateLogFiles() {
   for i := l.numFiles - 2; i > 0; i-- {
      _ = os.Rename(fmt.Sprint(l.logPath, ".", i), fmt.Sprint(l.logPath, ".", i+1))
   }
   _ = os.Rename(l.logPath, fmt.Sprint(l.logPath, ".", 1))
}

/*
 * logTargetFileDaily
 */
type logTargetFileDaily struct {
   logTarget
   logPath string
}

func NewLogTargetFileDaily(lvl LogLevel, logPath string) *logTargetFileDaily {
   return &logTargetFileDaily{logTarget{Level: lvl}, logPath}
}

func (l *logTargetFileDaily) Append(msg string) {
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

   // although Open/Close for each log decreases performance but
   // I want to save to disk when append is performed
   f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
   if err != nil {
      fmt.Printf("[ logTargetFileDaily:Append() ] file(\"%s\") open failed, error=%v",
         logPath, err.Error())
   }
   _, _ = f.WriteString(msg)
   _ = f.Close()
}

/*
 * logger
 */
type Logger struct {
   logTargets             []iLogTarget
   isReady                bool
   useColoredLogLevelName bool
}

func NewLogger(useColoredLogLevelName bool) *Logger {
   return &Logger{[]iLogTarget{}, false, useColoredLogLevelName}
}

func (l *Logger) AddTarget(target iLogTarget) {
   l.logTargets = append(l.logTargets, target)
   l.isReady = true
}

func (l *Logger) IsEnabled(lvl LogLevel) bool {
   if len(l.logTargets) <= 0 {
      return false
   }
   return l.logTargets[0].IsEnabled(lvl)
}

func (l *Logger) Trace(format string, v ...interface{}) {
   l.logFormat(TRACE, callDepth+1, fmt.Sprintf(format, v...))
}

func (l *Logger) Debug(format string, v ...interface{}) {
   l.logFormat(DEBUG, callDepth+1, fmt.Sprintf(format, v...))
}

func (l *Logger) Info(format string, v ...interface{}) {
   l.logFormat(INFO, callDepth+1, fmt.Sprintf(format, v...))
}

func (l *Logger) Warn(format string, v ...interface{}) {
   l.logFormat(WARN, callDepth+1, fmt.Sprintf(format, v...))
}

func (l *Logger) Error(format string, v ...interface{}) {
   l.logFormat(ERROR, callDepth+1, fmt.Sprintf(format, v...))
}

func (l *Logger) Fatal(format string, v ...interface{}) {
   l.logFormat(FATAL, callDepth+1, fmt.Sprintf(format, v...))
   os.Exit(1)
}

func (l *Logger) logFormat(lvl LogLevel, calldepth int, msg string) {
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
   _, file, line, ok := runtime.Caller(calldepth)
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

   for _, v := range l.logTargets {
      if v.IsEnabled(lvl) {
         v.Append(msg)
      }
   }
}

/*
 * logger Factory
 */
func GetLogger() *Logger {
   if singletonInstance == nil {
      fmt.Println("[ GetLoggerInstance ] Logger is not created, use InitLogger...")
      os.Exit(1)
   }
   return singletonInstance
}

/**
 * lvl : log level
 * limitSize : max log file size limit to rotate, only for RollSize
 * numFiles: number of log files to be maintained for rotating, only for RollSize
 * logPath : log path + name (/a/b/c/message)
 * rollType : log file rolling by Daily or Size
 * useColoredLogLevelName : colored log level name or not
 * force : force re-create singletonInstance when already exists
 */
func InitLogger(lvl LogLevel, limitSize int64, numFiles int, logPath string,
        rollType RollType, useColoredLogLevelName bool, force bool) *Logger {
   mutex.Lock()
   defer mutex.Unlock()

   if singletonInstance != nil && !force {
      return singletonInstance
   }

   singletonInstance = NewLogger(useColoredLogLevelName)

   // add Console
   singletonInstance.AddTarget(NewConsole(lvl))

   // add FileLog
   if rollType == RollDaily {
      singletonInstance.AddTarget(NewLogTargetFileDaily(lvl, logPath))
   } else {
      singletonInstance.AddTarget(NewLogTargetFileBySize(lvl, limitSize, numFiles, logPath))
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
