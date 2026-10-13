package role

import "sync"

// inbox reads a stream into an unbounded queue, so the peer's sends never
// wait on this side's own: two sides each blocked sending to the other
// would deadlock.
type inbox[T any] struct {
	ready chan struct{}

	mu    sync.Mutex
	queue []T
	err   error
}

// receive reads with next until it fails, on a goroutine in wg.
func receive[T any](wg *sync.WaitGroup, next func() (T, error)) *inbox[T] {
	in := &inbox[T]{ready: make(chan struct{}, 1)}
	wg.Go(func() {
		for {
			v, err := next()
			in.mu.Lock()
			if err != nil {
				in.err = err
			} else {
				in.queue = append(in.queue, v)
			}
			in.mu.Unlock()
			select {
			case in.ready <- struct{}{}:
			default:
			}
			if err != nil {
				return
			}
		}
	})
	return in
}

// take is everything read since the last take, then the read error that
// ended the stream, if it has.
func (in *inbox[T]) take() ([]T, error) {
	in.mu.Lock()
	defer in.mu.Unlock()
	q := in.queue
	in.queue = nil
	return q, in.err
}
