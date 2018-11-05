package logHelper

import (
	"github.com/go-kit/kit/log"
	"time"
)

func LogTimeMiddleware(logger log.Logger) log.Logger {
	logger = log.With(logger, "ts", log.TimestampFormat(time.Now().Local, "2006-01-02 15:04:05.000.000000"))
	logger = log.With(logger, "caller", log.DefaultCaller)
	return logger
}
