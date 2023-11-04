package server

import (
	"bytes"
	"github.com/mayuedong/unit"
	"golang.org/x/sys/unix"
	"io"
	"regexp"
	"time"
)

const _READ_BUF_SIZE_ = 1024 * 512

var proxyProtocolRegexp = regexp.MustCompile("PROXY TCP\\d ((\\d+\\.){3}\\d+)")

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
	buf := r.readBuf[:_READ_BUF_SIZE_]
	n, err := cli.read(buf)
	if nil != err && err != unix.EAGAIN && err != io.EOF {
		unit.Error("readn", n, err)
	}

	if n <= 0 {
		return
	}

	buf = buf[:n]
	for pos, count := bytes.IndexByte(buf, '\n'), 0; pos != -1; pos, count = bytes.IndexByte(buf, '\n'), count+1 {
		line := buf[:pos]
		//根据回车分割消息
		if len(buf) > pos+1 {
			buf = buf[pos+1:]
		} else {
			//TODO 下次轮询，不能设置nil
			buf = []byte{}
		}

		//decode成功处理数据
		if err = cli.CallbackRead(line, cli); nil == err {
			if 0 != len(cli.readCache) {
				cli.readCache = []byte{}
			}
			continue
		}

		//decode失败！TCP包的第一条消息？
		if count == 0 {
			//第一个TCP包decode失败，把消息扔掉
			if nil == cli.readCache {
				//PROXY TCP4 200.58.166.84 172.65.241.160 11829 44
				if ips := proxyProtocolRegexp.FindStringSubmatch(string(line)); len(ips) > 1 {
					cli.ip = ips[1]
				}
				cli.readCache = []byte{}
				continue
			}

			//第n个TCP包的第一条消息
			if 0 != len(cli.readCache) {
				//和上个包的尾部消息拼接再decode试试
				cli.readCache = append(cli.readCache, line...)
				//不管失败成功，都把cache清空，要把这个包的尾部残缺数据放进去
				cli.CallbackRead(cli.readCache, cli)
				cli.readCache = []byte{}
			}
		}
	}

	if 0 != len(buf) {
		//再次decode，有可能是完整的数据，不完整写入cache，和下一个包头组合
		if err = cli.CallbackRead(buf, cli); nil != err {
			cli.readCache = append(cli.readCache, buf...)
		}
	}
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
	for _, cli := range r.clients {
		cli.CallbackBroadcast(cli)
	}
}
