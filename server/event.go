package server

import (
	"github.com/mayuedong/unit"
)

type _EVENT_TYPE_ int

const (
	_NONE_EVENT_      _EVENT_TYPE_ = 0
	_ADD_EVENT_                    = 1
	_DEL_EVENT_                    = 2
	_READ_EVENT_                   = 4
	_WRITE_EVENT_                  = 8
	_BROADCAST_EVENT_              = 16
)

type event struct {
	mask   _EVENT_TYPE_
	client *Client
	loop   *eventLoop
}

// 减少锁的次数
type eventGroup struct {
	events []*event
}

func (r *eventGroup) Callback() {
	for _, v := range r.events {
		v.Callback()
	}
}

func (r *event) Callback() {
	//添加失败后会关闭套接字，这里不用处理错误
	if 0 != _ADD_EVENT_&r.mask {
		if err := r.loop.onAddClient(r.client); nil != err {
			unit.Error("AddCliEvent", err)
		} else {
			r.client.CallbackLogin()
		}
		return
	}

	if 0 != _BROADCAST_EVENT_&r.mask {
		r.loop.eventWorkers[r.client.posWorker].onBroadcast()
		return
	}

	//TODO 以下事件是epoll生产者创建的，为了避免用锁，创建时没有查找cli，消费者负责查找。
	worker := r.loop.eventWorkers[r.client.nfd%r.loop.numWorker]
	cli := worker.findClient(r.client.nfd)
	if nil == cli {
		unit.Info("ServNotFoundCli:", r.client.nfd, r.mask)
		return
	}

	if 0 != _DEL_EVENT_&r.mask {
		cli.CallbackLogout()
		if err := r.loop.onDelClient(worker, cli); nil != err {
			unit.Error("DelCliEvent", err)
		}
		return
	}

	if 0 != _READ_EVENT_&r.mask {
		r.loop.eventWorkers[cli.posWorker].read(cli)
	}
	//缓冲区由满变为不满时会触发写事件，上次未发送的数据在这里发出去
	if 0 != _WRITE_EVENT_&r.mask {
		cli.onWrite()
	}
}
