package logger_test

import (
   "fmt"
   "github.com/stretchr/testify/assert"
   "github.com/yslim/logger"
   "testing"
)

func Test(t *testing.T) {
   fmt.Println(logger.KB, logger.MB, logger.GB, logger.TB)
   fmt.Println(logger.ALL, logger.TRACE, logger.DEBUG, logger.INFO,
      logger.WARN, logger.ERROR, logger.FATAL, logger.OFF)

   loggerTest(t)
}

func loggerTest(t *testing.T) {
   // logger instance
   log := logger.NewLogger(false)
   log.AddTarget(logger.NewConsole(logger.INFO))
   log.AddTarget(logger.NewLogTargetFileBySize(logger.INFO, 10, 10, "/tmp/message"))

   assert.Equal(t, log.IsEnabled(logger.TRACE), false, "Log Level")
   assert.Equal(t, log.IsEnabled(logger.DEBUG), false, "Log Level")
   assert.Equal(t, log.IsEnabled(logger.INFO), true, "Log Level")
   assert.Equal(t, log.IsEnabled(logger.ERROR), true, "Log Level")

   log.Trace("[ Test ] trace message = %v", "\"Hello Logger\"")
   log.Debug("[ Test ] debug message = %v", "\"Hello Logger\"")
   log.Info("[ Test ] info message = %v", "\"Hello Logger\"")
   log.Warn("[ Test ] warn message = %v", "\"Hello Logger\"")
   log.Error("[ Test ] error message = %v", "\"Hello Logger\"")

   // global instance
   glog := logger.InitLogger(logger.ALL, 10, 10, "/tmp/message", logger.ROLL_SIZE, true, true)
   glog2 := logger.GetLogger();
   assert.Equal(t, glog, glog2)  // glog, glog2 are same pointer
   assert.NotEqual(t, glog, log) // glog and log are different

   glog.Trace("[ Test ] trace message = %v", "\"Hello Logger\"")
   glog.Debug("[ Test ] debug message = %v", "\"Hello Logger\"")
   glog.Info("[ Test ] info message = %v", "\"Hello Logger\"")
   glog.Warn("[ Test ] warn message = %v", "\"Hello Logger\"")
   glog.Error("[ Test ] error message = %v", "\"Hello Logger\"")
}

func BenchmarkLogger(b *testing.B) {
   b.Run("yslim.Logger.Parallel", func(b *testing.B) {
      log := logger.InitLogger(logger.ALL, 1024*1024*10, 10, "/Volumes/MGTEC/Torrent/message",
         logger.ROLL_SIZE, true, true)
      b.ResetTimer()
      b.RunParallel(func(pb *testing.PB) {
         for pb.Next() {
            log.Info(getMessage())
         }
      })
   })
   b.Run("yslim.Logger", func(b *testing.B) {
      log := logger.InitLogger(logger.ALL, 1024*1024*10, 10, "/Volumes/MGTEC/Torrent/message",
         logger.ROLL_SIZE, true, true)
      b.ResetTimer()
      for n := 0; n < b.N; n++ {
         log.Debug(getMessage())
      }
   })
}

var (
   _messages = fakeMessages(1000)
   _iter     = 0
)

func fakeMessages(n int) []string {
   messages := make([]string, n)
   for i := range messages {
      messages[i] = fmt.Sprintf("Test logging, but use a somewhat realistic message length. (#%v)", i)
   }
   return messages
}

func getMessage() string {
   _iter = (_iter + 1) % 1000
   return _messages[_iter]
}
