package unboundedchannel

import "context"

// New returns a pair of channels (in, out) that implement an unbounded FIFO using a slice as the buffer.
// Writes to in never block; reads from out block only if the buffer is empty and in is not closed.
// The caller must close in to eventually close out, and must drain out to fully release resources.
// Failing to close in or drain out after closing in leaks a goroutine.
func New[T any]() (chan<- T, <-chan T) {
	return NewWithContext[T](context.Background())
}

// NewWithContext returns a pair of channels (in, out) that implement an unbounded FIFO using a slice as the buffer.
// Writes to in never block (unless context is done); reads from out block only if the buffer is empty and in is not closed.
// The provided ctx is used to cancel any pending operations and terminate buffering early.
// The caller must either cancel the context or close in to eventually close out, and must drain out to fully release resources.
// Failing to close the context or failing to either close in or drain out leaks a goroutine.
//
// When writing messages to in, callers should select on ctx.Done() to avoid
// blocking after cancellation. For example:
//
//	select {
//	case in <- msg:
//	    // Write succeeded
//	case <-ctx.Done():
//	    // Write dropped due to context cancellation
//	}
func NewWithContext[T any](ctx context.Context) (chan<- T, <-chan T) {
	in := make(chan T)
	out := make(chan T)

	// Start buffering
	go buffer(ctx, in, out)

	return in, out
}

func buffer[T any](ctx context.Context, in <-chan T, out chan<- T) {
	defer close(out)

	var buffer []T

	// Outer loop only adds to buffer
loop:
	for {
		select {
		case t, ok := <-in:
			if !ok {
				return // Buffer is empty here
			}

			buffer = append(buffer, t)

			// Inner loop both adds to buffer and writes to out
			for len(buffer) > 0 {
				select {
				case t, ok := <-in:
					// When in is closed, exit loop
					if !ok {
						break loop
					}

					buffer = append(buffer, t)
				case out <- buffer[0]:
					buffer[0] = *new(T)
					buffer = buffer[1:]
				case <-ctx.Done():
					return
				}
			}

			// Release buffer everytime it's emptied
			buffer = nil
		case <-ctx.Done():
			return
		}
	}

	// Write out rest of the messages to out before exit
	for _, t := range buffer {
		select {
		case out <- t:
		case <-ctx.Done():
			return
		}
	}
}
