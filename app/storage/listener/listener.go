package listener

import "redis-go/app/resp"

type Listener struct {
	Channel chan resp.Resp
	Id      int
}

type Listeners struct {
	Listeners []Listener
}
