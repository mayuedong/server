package api

type AnyClient interface {
	CallbackRead([]byte, AnyWeakClient) error
	CallbackBroadcast(AnyWeakClient)
	CallbackLogin()
	CallbackLogout()
}
type AnyWeakClient interface {
	Write([]byte)
	Fd() int
	Ip() string
}

type AnyWeakServiceClientele interface {
	DelClient(AnyWeakClient)
}
type AnyWeakServiceBroadcaster interface {
	Broadcast()
}

type AnyBroadcaster interface {
	Close()
}
type AnyWeakBroadcaster interface {
	GetNetHeight() uint64
}
