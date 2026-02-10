package systems

type Filter[T any] func(target T) bool

type Broadcaster[T any, M any] struct {
	iterator func(yield func(T) bool)
	send     func(target T, message M)
}

func NewBroadcaster[T any, M any](iterator func(yield func(T) bool), send func(target T, message M)) *Broadcaster[T, M] {
	return &Broadcaster[T, M]{
		iterator: iterator,
		send:     send,
	}
}

func (b *Broadcaster[T, M]) Broadcast(message M, filters ...Filter[T]) {
	b.iterator(func(target T) bool {
		for _, filter := range filters {
			if !filter(target) {
				return true
			}
		}
		b.send(target, message)

		return true
	})
}

func NotSender[T comparable](sender T) Filter[T] {
	return func(target T) bool {
		return target != sender
	}
}
