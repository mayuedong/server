package server

import (
	"errors"
	"fmt"
	"github.com/mayuedong/server/api"
	"github.com/mayuedong/unit"
	"golang.org/x/sys/unix"
	"sync"
	"sync/atomic"
)

type Server struct {
	fd          int
	running     atomic.Bool
	poll        *eventLoop
	onceClose   sync.Once
	broadcaster api.AnyBroadcaster
}

func Listen(port int, worker int,
	newClientele func(api.AnyWeakServiceClientele, api.AnyWeakBroadcaster) api.AnyClient,
	NewBroadcaster func(broadcaster api.AnyWeakServiceBroadcaster) (api.AnyBroadcaster, api.AnyWeakBroadcaster, error)) (*Server, error) {
	//创建广播者
	r := &Server{}
	broadcaster, weakBroadcaster, err := NewBroadcaster(r)
	if nil != err {
		return nil, err
	}
	r.broadcaster = broadcaster

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
	//监听，队列长度设置为256
	if err = unix.Listen(fd, 256); nil == err {
		r.fd = fd
		//创建事务调度器，管理客户端套接字
		if r.poll, err = newEventLoop(worker); nil != err {
			goto ERROR
		}
		//等待连接到来
		go func() {
			r.running.Store(true)
			for r.running.Load() {
				if e := r.accept(newClientele(r, weakBroadcaster)); nil != e {
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
	client := &Client{nfd: nfd, AnyClient: cli, readCache: nil}
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
		client.ip = fmt.Sprintf("%d.%d.%d.%d", addr.Addr[0], addr.Addr[1], addr.Addr[2], addr.Addr[3])
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
	r.onceClose.Do(func() {
		r.running.Store(false)
		//关闭所有的客户端
		r.poll.close()
		//关闭广播者
		r.broadcaster.Close()
		//关闭服务端套接字
		unix.Close(r.fd)
		unit.Info("ServerStopped")
	})

}

// 广播者调用
func (r *Server) Broadcast() {
	if r.running.Load() {
		r.poll.broadcastClient()
	}
}

// 断开连接
func (r *Server) DelClient(client api.AnyWeakClient) {
	if r.running.Load() {
		if cli, ok := client.(*Client); ok && nil != cli {
			r.poll.eventWorkers[cli.posWorker].pushEvent(&event{mask: _DEL_EVENT_, client: cli, loop: r.poll})
		}
	}
}
