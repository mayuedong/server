package clientele

import (
	"github.com/mayuedong/server/api"
	"time"
)

const (
	_ENTER_ = '\n'
)

type Clientele struct {
	weakServ        api.AnyWeakServiceClientele
	weakBroadcaster api.AnyWeakBroadcaster
	ip              string
	loginTime       time.Time
	logoutTime      time.Time
	lastReadCache   []byte
}

// 钱包接口用于获取job
// server接口用于断开矿机连接【无法登录时踢掉】
func NewClientele(anyService api.AnyWeakServiceClientele, anyWallet api.AnyWeakBroadcaster) api.AnyClient {
	r := &Clientele{weakServ: anyService, weakBroadcaster: anyWallet, lastReadCache: nil}
	return r
}
