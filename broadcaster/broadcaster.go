package broadcaster

import (
	"github.com/mayuedong/server/api"
)

type Broadcaster struct {
	broadcastOperation api.AnyWeakServiceBroadcaster
}

func NewBroadcaster(operation api.AnyWeakServiceBroadcaster) (api.AnyBroadcaster, api.AnyWeakBroadcaster, error) {
	r := &Broadcaster{
		broadcastOperation: operation,
	}
	return r, r, nil
}

func (r *Broadcaster) Close() {

}

func (r *Broadcaster) GetNetHeight() uint64 {
	return 0
}
