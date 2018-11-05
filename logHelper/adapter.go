package logHelper

import (
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-sql-driver/mysql"
)

type dbLogger struct {
	logger log.Logger
}

func (self dbLogger) Print(v ...interface{}) {
	self.logger.Log("text", fmt.Sprint(v))
}

func MysqlLogger(logger log.Logger) mysql.Logger {
	return dbLogger{logger}
}
