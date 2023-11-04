package server

import (
	"errors"
	"github.com/mayuedong/unit"
	"golang.org/x/sys/unix"
	"github.com/mayuedong/server/api"
	"sync/atomic"
)

type Server struct {
	fd          int
	running     atomic.Bool
	poll        *eventLoop
	broadcaster api.AnyBroadcaster
}

func Listen(port int, newClientele func(api.AnyWeakServiceClientele, api.AnyWeakBroadcaster) api.AnyClient,
	NewBroadcaster func(broadcaster api.AnyWeakServiceBroadcaster) (api.AnyBroadcaster, api.AnyWeakBroadcaster, error)) (*Server, error) {
	//启动钱包，生成job template
	r := &Server{}
	wallet, weakWallet, err := NewBroadcaster(r)
	if nil != err {
		return nil, err
	}
	r.broadcaster = wallet

	//创建套接字
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, unix.IPPROTO_TCP)
	if nil != err {
		return nil, err
	}
	//设置地址重用
	if err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); nil != err {
		goto ERROR
	}
	//绑定端口
	if err = unix.Bind(fd, &unix.SockaddrInet4{Port: port}); nil != err {
		goto ERROR
	}
	//监听，队列长度设置为512
	if err = unix.Listen(fd, 256); nil == err {
		r.fd = fd
		//创建事务调度器，管理客户端套接字
		if r.poll, err = newEventLoop(8); nil != err {
			goto ERROR
		}
		//等待连接到来
		go func() {
			r.running.Store(true)
			for r.running.Load() {
				if e := r.accept(newClientele(r, weakWallet)); nil != e {
					unit.Error("accept socket", e)
				}
			}
		}()
		return r, nil
	}

ERROR:
	unix.Close(fd)
	return nil, err
}

func (r *Server) accept(cli api.AnyClient) error {
	//接受客户端连接
	nfd, sa, err := unix.Accept(r.fd)
	if nil != err {
		return err
	}
	client := &Client{nfd: nfd, AnyClient: cli}
	//设置为非阻塞
	if err = unix.SetNonblock(nfd, true); nil != err {
		goto ERROR
	}
	//设置保持连接
	if err = unix.SetsockoptInt(nfd, unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1); nil != err {
		goto ERROR
	}
	//设置非等待
	if err = unix.SetsockoptInt(nfd, unix.IPPROTO_TCP, unix.TCP_NODELAY, 1); nil != err {
		goto ERROR
	}
	//托管客户端连接
	if addr, ok := sa.(*unix.SockaddrInet4); ok {
		client.addr = addr
		err = r.poll.addClient(client)
	} else {
		err = errors.New("OnlySupportTCP4")
	}

ERROR:
	if nil != err {
		unix.Close(nfd)
	}
	return err
}

func (r *Server) Close() {
	if !r.running.Load() {
		return
	}
	r.running.Store(false)
	//关闭所有的客户端
	r.poll.close()
	//关闭钱包
	r.broadcaster.Close()
	//关闭服务端套接字
	unix.Close(r.fd)
	unit.Info("ServerStopped")
}

// 获取到新高度的时候广播
func (r *Server) Broadcast() {
	if r.running.Load() {
		r.poll.broadcastClient()
	}
}

// 登录失败的时候断开连接
func (r *Server) DelClient(client api.AnyWeakClient) {
	if cli, ok := client.(*Client); ok && nil != cli {
		worker := r.poll.eventWorkers[cli.posWorker]
		worker.pushEvent(&event{mask: _DEL_EVENT_, client: cli, loop: r.poll})
	}
}
