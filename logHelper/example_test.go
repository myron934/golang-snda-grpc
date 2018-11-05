package logHelper_test

import (
	"fmt"
	"github.com/go-kit/kit/log"
	"local/sndaRpc/logHelper"
	"math/rand"
	"sync"
	"time"
)

func ExampleBase() {
	name := "all"
	_, err := logHelper.Register(name, "all.log", "debug")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	logHelper.Debug(name).Log("name", "alice", "age", 1)
}

func ExampleWith() {
	name := "all"
	_, err := logHelper.Register(name, "all.log", "debug")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var wg sync.WaitGroup
	RunTask := func(id int, logger log.Logger) {
		// 月(1) 日(2) 时(3) 分(4) 秒(5) 年(6)
		logger = log.With(logger, "time", log.TimestampFormat(time.Now().Local, "2006-01-02 15:04:05.000.000000"), "contextualTaskID", id)
		logger.Log("taskID", id, "event", "starting task")
		time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)

		logger.Log("taskID", id, "event", "task complete")
		wg.Done()
	}

	wg.Add(4)

	go RunTask(1, logHelper.Debug(name))
	go RunTask(2, logHelper.Info(name))
	go RunTask(3, logHelper.Warn(name))
	go RunTask(4, logHelper.Error(name))

	wg.Wait()
}
