package unboundedchannel

import "context"

// Unbounded channel
// Returns two channels, in and out
// Writing into in buffers data into a slice
// Reading from out reads from the buffer or blocks if no data is available
//
// Input channel must be closed and output channel must be drained
// after use for resources to be released
// Out channel is closed after in has been closed an buffer emptied
func New[T any]() (chan<- T, <-chan T) {
	return NewWithContext[T](context.Background())
}

// With context
// When writing messages to in, remember to check that the context is not done
// This can be done for example like this:
// ```
// select {
// case in <- msg:
//     // Write success
// case <-ctx.Done():
//     // Write dropped
// }
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
