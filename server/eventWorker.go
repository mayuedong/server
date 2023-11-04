package server

import (
	"github.com/mayuedong/unit"
	"golang.org/x/sys/unix"
	"io"
	"time"
)

const _READ_BUF_SIZE_ = 1024 * 512

// one worker one callBacker
type eventWorker struct {
	count     uint8
	worker    *unit.WorkerPool
	clients   map[int]*Client
	name      string
	readBuf   []byte
	startTime time.Time
}

// main call
func newEventWorker(name string) *eventWorker {
	return &eventWorker{
		name:    name,
		clients: make(map[int]*Client),
		readBuf: make([]byte, _READ_BUF_SIZE_),
		worker:  unit.NewWorkerPool(-1, 1),
	}
}

func (r *eventWorker) read(cli *Client) {
	if r.count%200 == 0 {
		r.startTime = time.Now()
	}
	buf := r.readBuf[:_READ_BUF_SIZE_]
	n, err := cli.read(buf)

	if n > 0 {
		cli.CallbackRead(buf[:n], cli)
	}

	if nil != err && err != unix.EAGAIN && err != io.EOF {
		unit.Error("readn", n, err)
	}
	if r.count%200 == 0 {
		unit.Info(r.name, "readCost", time.Now().Sub(r.startTime))
	}
	r.count++
}

// main call
func (r *eventWorker) close() {
	//这里会阻塞，将队列中的数据消费完，才会退出
	r.worker.Close()
	for _, cli := range r.clients {
		cli.close()
	}
	unit.Info("EventWorker closed", r.name, len(r.clients))
}

func (r *eventWorker) pushEvents(even *eventGroup) {
	r.worker.Push(even)
}

// eventLoop call
func (r *eventWorker) pushEvent(even *event) {
	r.worker.Push(even)
}
func (r *eventWorker) addEvent(even *event) {
	r.worker.Add(even)
}

func (r *eventWorker) addClient(cli *Client) {
	r.clients[cli.nfd] = cli
}
func (r *eventWorker) delClient(cli *Client) {
	delete(r.clients, cli.nfd)
}
func (r *eventWorker) findClient(nfd int) *Client {
	return r.clients[nfd]
}

func (r *eventWorker) onBroadcast() {
	if r.count%200 == 0 && 0 != len(r.clients) {
		r.startTime = time.Now()
	}
	for _, cli := range r.clients {
		cli.CallbackBroadcast(cli)
	}
	if r.count%200 == 0 && 0 != len(r.clients) {
		unit.Info(r.name, "broadcast", len(r.clients), "cost", time.Now().Sub(r.startTime))
	}
	r.count++
}
