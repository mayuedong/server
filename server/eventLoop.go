package server

import (
	"errors"
	"fmt"
	"github.com/mayuedong/unit"
	"golang.org/x/sys/unix"
	"sync/atomic"
)

const _EPOLL_EVENT_SIZE_ = 1024

type eventLoop struct {
	efd            int
	numWorker      int
	running        atomic.Bool
	producerEvents []unix.EpollEvent
	eventWorkers   []*eventWorker
}

func newEventLoop(threads int) (r *eventLoop, err error) {
	//创建EPOLL
	efd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if nil != err {
		return nil, err
	}

	r = &eventLoop{
		efd:            efd,
		numWorker:      threads,
		producerEvents: make([]unix.EpollEvent, _EPOLL_EVENT_SIZE_),
		eventWorkers:   make([]*eventWorker, threads)}

	//创建线程池
	for i := 0; i < len(r.eventWorkers); i++ {
		r.eventWorkers[i] = newEventWorker(fmt.Sprintf("worker%d", i))
	}

	go func() {
		r.running.Store(true)
		for r.running.Load() {
			if e := r.wait(); nil != e && e != unix.EINTR {
				unit.Error("epollWaitErr", e)
			}
		}
	}()
	return
}

// main call
func (r *eventLoop) close() {
	if !r.running.Load() {
		return
	}
	r.running.Store(false)
	//关闭线程池
	for i := 0; i < len(r.eventWorkers); i++ {
		r.eventWorkers[i].close()
	}
	//关闭EPOLL
	unix.Close(r.efd)
	unit.Info("EventLoop closed")
}

// epoll 生产者
func (r *eventLoop) wait() error {
	r.producerEvents = r.producerEvents[:_EPOLL_EVENT_SIZE_]
	n, err := unix.EpollWait(r.efd, r.producerEvents, -1)
	if nil != err {
		return err
	}
	consumerEvents := make(map[int]*eventGroup)
	for i := 0; i < n; i++ {
		mask := _NONE_EVENT_
		nfd := int(r.producerEvents[i].Fd)
		kind := r.producerEvents[i].Events

		if 0 != kind&unix.EPOLLIN {
			mask |= _READ_EVENT_
		}
		if 0 != kind&unix.EPOLLRDHUP {
			mask |= _DEL_EVENT_
		}
		if 0 != kind&unix.EPOLLOUT {
			mask |= _WRITE_EVENT_
		}
		//TODO 这几个事件，client信息不全。这里没有查找，消费的时候查找一下，避免用锁。
		pEvent := &event{mask: mask, client: &Client{nfd: nfd}, loop: r}
		if group, ok := consumerEvents[nfd%r.numWorker]; !ok {
			consumerEvents[nfd%r.numWorker] = &eventGroup{events: []*event{pEvent}}
		} else {
			group.events = append(group.events, pEvent)
			consumerEvents[nfd%r.numWorker] = group
		}
	}
	//将多个客户端组合成一个任务，减少锁使用的次数
	for k, v := range consumerEvents {
		r.eventWorkers[k].pushEvents(v)
	}
	return err
}

// broadcaster call 给所有的线程注册广播事件
func (r *eventLoop) broadcastClient() {
	if r.running.Load() {
		for j := 0; j < len(r.eventWorkers); j++ {
			r.eventWorkers[j].addEvent(&event{mask: _BROADCAST_EVENT_, client: &Client{posWorker: j}, loop: r})
		}
	}
}

// main call 根据fd，添加到指定的线程池中去
func (r *eventLoop) addClient(cli *Client) error {
	if !r.running.Load() {
		return errors.New("STOPPED")
	}
	cli.posWorker = cli.nfd % r.numWorker
	cli.writeCache.Init(-1)
	r.eventWorkers[cli.posWorker].pushEvent(&event{mask: _ADD_EVENT_, client: cli, loop: r})
	return nil
}

// TODO 顺序不能颠倒，先缓存，再注册
func (r *eventLoop) onAddClient(cli *Client) error {
	worker := r.eventWorkers[cli.posWorker]
	worker.addClient(cli)

	if err := unix.EpollCtl(r.efd, unix.EPOLL_CTL_ADD, cli.nfd, &unix.EpollEvent{

		Events: unix.EPOLLET | unix.EPOLLRDHUP | unix.EPOLLOUT | unix.EPOLLIN, Fd: int32(cli.nfd)}); nil != err {
		worker.delClient(cli)
		cli.close()
		return err
	}
	return nil
}

// TODO 顺序不能颠倒，和添加相反
func (r *eventLoop) onDelClient(worker *eventWorker, cli *Client) (err error) {
	if err = unix.EpollCtl(r.efd, unix.EPOLL_CTL_DEL, cli.nfd, nil); nil != err {
		unit.Info(worker.name, "delCli", cli.nfd, err)
	}
	worker.delClient(cli)
	cli.close()
	return
}
