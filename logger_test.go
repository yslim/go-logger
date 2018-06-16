package logger_test

import (
	"fmt"
   "github.com/magiconair/properties/assert"
   "github.com/yslim/logger"
	"testing"
)

func Test(t *testing.T) {
   fmt.Println(logger.KB, logger.MB, logger.GB, logger.TB)
   fmt.Println(logger.ALL, logger.TRACE, logger.DEBUG, logger.INFO,
      logger.WARN, logger.ERROR, logger.FATAL, logger.OFF)

   //var ilt logger.ILogTarget
   //ilt = logger.New(logger.INFO)
   //fmt.Println(ilt.IsEnabled(logger.ERROR))

   log := logger.GetLogger(logger.ALL, 10, 10, "/tmp/message", logger.ROLL_SIZE)
   assert.Equal(t, log.IsEnabled(logger.TRACE), false, "Log Level")
   assert.Equal(t, log.IsEnabled(logger.DEBUG), false, "Log Level")
   assert.Equal(t, log.IsEnabled(logger.INFO), true, "Log Level")
   assert.Equal(t, log.IsEnabled(logger.ERROR), true, "Log Level")

   log.Trace("[ Test ] trace message = %v", "\"Hello Logger\"")
   log.Debug("[ Test ] debug message = %v", "\"Hello Logger\"")
   log.Info("[ Test ] info message = %v", "\"Hello Logger\"")
   log.Warn("[ Test ] warn message = %v", "\"Hello Logger\"")
   log.Error("[ Test ] error message = %v", "\"Hello Logger\"")
   log.Fatal("[ Test ] fatal message = %v", "\"우리나라 대한민국\"")
}