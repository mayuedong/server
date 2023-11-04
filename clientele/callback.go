package clientele

import (
	"bytes"
	"encoding/json"
	"github.com/mayuedong/unit"
	"regexp"
	"github.com/mayuedong/server/api"
	"time"
)

func (r *Clientele) CallbackLogin() {
	r.loginTime = time.Now()
}

func (r *Clientele) CallbackLogout() {
	//没有完成登录校验,不走退出流程

}

func (r *Clientele) CallbackBroadcast(cli api.AnyWeakClient) {
	//尚未登录校验，不下发任务

}

// TODO 入参不能跨线程使用，多个client复用buf，不能引用
var proxyProtocolRegexp = regexp.MustCompile("PROXY TCP\\d ((\\d+\\.){3}\\d+)")

func (r *Clientele) CallbackRead(buf []byte, cli api.AnyWeakClient) {
	event := &jsonEvent{}
	for pos, count := bytes.Index(buf, []byte{_ENTER_}), 0; pos != -1; pos, count = bytes.Index(buf, []byte{_ENTER_}), count+1 {
		line := buf[:pos]
		//根据回车分割消息
		if len(buf) > pos+1 {
			buf = buf[pos+1:]
		} else {
			//TODO 下次轮询，不能设置nil
			buf = []byte{}
		}

		//消息全部都是空格
		line = bytes.TrimSpace(line)
		if 0 == len(line) {
			continue
		}

		err := json.Unmarshal(line, event)
		//decode成功处理数据
		if nil == err {
			if 0 != len(r.lastReadCache) {
				r.lastReadCache = []byte{}
			}
			r.handlerMessage(cli, event)
			continue
		}
		//decode失败！TCP包的第一条消息？
		if count == 0 {
			//第一个TCP包decode失败，把消息扔掉
			if nil == r.lastReadCache {
				//PROXY TCP4 200.58.166.84 172.65.241.160 11829 44
				strLine := string(line)
				if ips := proxyProtocolRegexp.FindStringSubmatch(strLine); len(ips) > 1 {
					r.ip = ips[1]
				} else {
					unit.Warn("decodeFirstMsg", err, "line", strLine)
				}
				r.lastReadCache = []byte{}
				continue
			}
			//第n个TCP包的第一条消息
			if 0 != len(r.lastReadCache) {
				//和上个包的尾部消息拼接再decode试试
				r.lastReadCache = append(r.lastReadCache, line...)
				err = json.Unmarshal(r.lastReadCache, event)
				if nil != err {
					unit.Warn("decodeCache", err, "cache", string(r.lastReadCache))
				} else {
					r.handlerMessage(cli, event)
				}
				//不管失败成功，都把cache清空，要把这个包的尾部残缺数据放进去
				r.lastReadCache = []byte{}
			}
		} else {
			//不是TCP包的第一条数据
			unit.Error("decodeMsg", err, "line", count, string(line))
			continue
		}
	}

	if 0 != len(buf) {
		//再次decode，有可能是完整的数据，不完整写入cache，和下一个包头组合
		if err := json.Unmarshal(buf, event); nil != err {
			r.lastReadCache = append(r.lastReadCache, buf...)
		} else {
			r.handlerMessage(cli, event)
		}
	}
}
func (r *Clientele) handlerMessage(cli api.AnyWeakClient, event *jsonEvent) {

}
