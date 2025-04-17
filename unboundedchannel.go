package unboundedchannel

// Unbounded channel
// Returns two channels, in and out
// Writing into in buffers data into a slice
// Reading from out reads from the buffer or blocks if no data is available
//
// Input channel must be closed and output channel must be drained
// after use for resources to be released
// Out channel is closed after in has been closed an buffer emptied
func New[T any]() (chan<- T, <-chan T) {
	in := make(chan T)
	out := make(chan T)

	// Start buffering
	go buffer(in, out)

	return in, out
}

func buffer[T any](in <-chan T, out chan<- T) {
	defer close(out)

	var buffer []T

	// Outer loop only adds to buffer
loop:
	for t := range in {
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
			}
		}

		// Release buffer everytime it's emptied
		buffer = nil
	}

	// Write out rest of the messages to out before exit
	for _, t := range buffer {
		out <- t
	}
}
