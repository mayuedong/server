package main

import (
	"github.com/mayuedong/unit"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"os/signal"
	"github.com/mayuedong/server/broadcaster"
	"github.com/mayuedong/server/clientele"
	"github.com/mayuedong/server/server"
)

func main() {
	//初始化异步日志
	if err := unit.NewLog("./logs/log", unit.MONTH, unit.DAY); nil != err {
		log.Fatalln(err)
	}
	//defer unit.Close()

	//启动服务
	serv, err := server.Listen(9031, clientele.NewClientele, broadcaster.NewBroadcaster)
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
