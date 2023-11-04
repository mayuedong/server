package api

type AnyClient interface {
	CallbackRead([]byte, AnyWeakClient)
	CallbackBroadcast(AnyWeakClient)
	CallbackLogin()
	CallbackLogout()
}
type AnyWeakClient interface {
	Write([]byte)
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
