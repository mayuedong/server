package clientele

import (
	"encoding/json"
	"github.com/mayuedong/server/api"
	"regexp"
	"time"
)

type Clientele struct {
	weakServ        api.AnyWeakServiceClientele //断开连接
	weakBroadcaster api.AnyWeakBroadcaster      //与广播者交互
	loginTime       time.Time
}

func NewClientele(anyService api.AnyWeakServiceClientele, anyWallet api.AnyWeakBroadcaster) api.AnyClient {
	r := &Clientele{weakServ: anyService, weakBroadcaster: anyWallet}
	return r
}

func (r *Clientele) CallbackLogin() {
	r.loginTime = time.Now()
}

func (r *Clientele) CallbackLogout() {

}

func (r *Clientele) CallbackBroadcast(cli api.AnyWeakClient) {

}

// TODO 入参不能跨线程使用，多个client复用buf，不能引用
var proxyProtocolRegexp = regexp.MustCompile("PROXY TCP\\d ((\\d+\\.){3}\\d+)")

func (r *Clientele) CallbackRead(buf []byte, cli api.AnyWeakClient) (err error) {
	event := &jsonEvent{}
	if err = json.Unmarshal(buf, event); nil == err {
		r.handlerMessage(cli, event)
	}
	return err
}

type jsonEvent struct {
}

func (r *Clientele) handlerMessage(cli api.AnyWeakClient, event *jsonEvent) {

}
