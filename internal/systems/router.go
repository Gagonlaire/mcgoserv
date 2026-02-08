package systems

type Handler[C any, D any] func(context C, data D)

type DoubleRouter[R comparable, S comparable, C any, D any] struct {
	handlers map[R]map[S]Handler[C, D]
}

// NewDoubleRouter creates a new DoubleRouter with two levels of keys
func NewDoubleRouter[R comparable, S comparable, C any, D any]() *DoubleRouter[R, S, C, D] {
	return &DoubleRouter[R, S, C, D]{
		handlers: make(map[R]map[S]Handler[C, D]),
	}
}

func (r *DoubleRouter[R, S, C, D]) RegisterHandler(key1 R, key2 S, handler Handler[C, D]) {
	if _, ok := r.handlers[key1]; !ok {
		r.handlers[key1] = make(map[S]Handler[C, D])
	}
	r.handlers[key1][key2] = handler
}

func (r *DoubleRouter[R, S, C, D]) Handle(key1 R, key2 S, context C, data D) {
	if stateHandlers, ok := r.handlers[key1]; ok {
		if handler, ok := stateHandlers[key2]; ok {
			handler(context, data)
		}
	}
}
