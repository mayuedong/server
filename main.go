package main

import (
	"github.com/mayuedong/server/broadcaster"
	"github.com/mayuedong/server/clientele"
	"github.com/mayuedong/server/server"
	"github.com/mayuedong/unit"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"os/signal"
	"runtime"
)

func main() {
	//初始化异步日志
	logger, err := unit.NewLogger("./logs/log", unit.MONTH, unit.DAY)
	if nil != err {
		log.Fatalln(err)
	}
	defer logger.Close()

	//启动服务
	serv, err := server.Listen(9031, runtime.NumCPU(), clientele.NewClientele, broadcaster.NewBroadcaster)
	if nil != err {
		unit.Error("init server", err)
		return
	}
	defer serv.Close()

	stop := make(chan os.Signal)
	signal.Notify(stop, unix.SIGHUP, unix.SIGINT, unix.SIGTERM, unix.SIGQUIT)
	unit.Info("CatchSignal:", <-stop)
	close(stop)
}
