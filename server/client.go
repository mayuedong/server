package server

import (
	"github.com/mayuedong/server/api"
	"github.com/mayuedong/unit"
	"golang.org/x/sys/unix"
	"io"
)

type Client struct {
	nfd       int
	posWorker int
	api.AnyClient
	ip         string
	readCache  []byte
	writeCache unit.List[*[]byte]
}

func (r *Client) Fd() int {
	return r.nfd
}
func (r *Client) Ip() string {
	return r.ip
}

// TODO 做成接口，只能消费事件的线程调用，不能跨线程
func (r *Client) Write(p []byte) {
	//有历史数据，添加到队尾，按顺序发送
	r.writeCache.Push(&p)
	//缓冲区由满变为不满时会触发写事件，这里得手动调用一下
	r.onWrite()
}

func (r *Client) write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	//忽略系统中断，一直写
	n, err := unix.Write(r.nfd, p)
	for err == unix.EINTR {
		n, err = unix.Write(r.nfd, p)
	}

	if n == 0 && nil == err {
		return n, io.ErrUnexpectedEOF
	}
	return n, err
}

func (r *Client) onWrite() {
	for num := r.writeCache.Count(); num > 0; num = r.writeCache.Count() {
		p, ok := r.writeCache.PopHead()
		if !ok {
			continue
		}
		n, err := r.write(*p)
		//缓冲区已经满了，将剩下的数据缓存起来下次发送。
		if n < len(*p) {
			if err == unix.EAGAIN || nil == err || err == io.ErrUnexpectedEOF {
				if n < 0 {
					n = 0
				}
				leftover := (*p)[n:]
				//添加到队列头部，下次写保证数据完整
				r.writeCache.Add(&leftover)
				return
			}

			//其他错误日志记录，不再写到缓存【这次发送不成功，下次也不一定能成功】
			unit.Error("servWriten", n, err, len(*p))
		}
	}
}

func (r *Client) close() {
	//不判断返回错误，关闭失败你能咋？
	unix.Close(r.nfd)
}

func (r *Client) read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	//忽略系统中断，一直读
	n, err := unix.Read(r.nfd, p)
	for err == unix.EINTR {
		n, err = unix.Read(r.nfd, p)
	}
	//EOF 不用关闭套接字，EPOLL 注册了 EPOLLRDHUP， 在内核2.6.17及其之后的版本，对方关闭会触发删除事件
	if n == 0 && nil == err {
		return n, io.EOF
	}

	return n, err
}
