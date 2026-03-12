package server

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type Handler struct {
	Decode  func(pkt *packet.InboundPacket) (any, error)
	Process func(conn *Connection, pkt any)
	Ticked  bool
}

type Router struct {
	handlers [][]Handler
}

func NewRouter(maxStates int) *Router {
	return &Router{
		handlers: make([][]Handler, maxStates),
	}
}

func (r *Router) ensureCapacity(state mc.State, id int) {
	if int(state) >= len(r.handlers) {
		newHandlers := make([][]Handler, state+1)
		copy(newHandlers, r.handlers)
		r.handlers = newHandlers
	}

	if r.handlers[state] == nil {
		r.handlers[state] = make([]Handler, id+1)
	} else if id >= len(r.handlers[state]) {
		newSize := max(id+1, len(r.handlers[state])*2)
		newInner := make([]Handler, newSize)
		copy(newInner, r.handlers[state])
		r.handlers[state] = newInner
	}
}

func RegisterIgnored(r *Router, state mc.State, id int) {
	r.ensureCapacity(state, id)
	r.handlers[state][id] = Handler{
		Process: func(*Connection, any) {},
	}
}

func RegisterRaw(
	r *Router,
	state mc.State,
	id int,
	ticked bool,
	process func(conn *Connection, pkt *packet.InboundPacket),
) {
	r.ensureCapacity(state, id)
	r.handlers[state][id] = Handler{
		Ticked: ticked,
		Process: func(conn *Connection, p any) {
			process(conn, p.(*packet.InboundPacket))
		},
	}
}

func RegisterTyped[P any](
	r *Router,
	state mc.State,
	id int,
	ticked bool,
	decode func(pkt *packet.InboundPacket) (P, error),
	process func(conn *Connection, p P),
) {
	r.ensureCapacity(state, id)
	r.handlers[state][id] = Handler{
		Decode: func(raw *packet.InboundPacket) (any, error) {
			return decode(raw)
		},
		Ticked: ticked,
		Process: func(conn *Connection, p any) {
			process(conn, p.(P))
		},
	}
}

func (r *Router) Get(state mc.State, id int) (Handler, bool) {
	inner := r.handlers[state]
	if id < 0 || id >= len(inner) {
		return Handler{}, false
	}

	handler := inner[id]
	if handler.Process == nil {
		return Handler{}, false
	}

	return handler, true
}
