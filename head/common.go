package head

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"sync"
)

// ReadLogger continuously reads from an io.Reader, and blurts to the specified log.Logger
func ReadLogger(reader io.Reader, writer *log.Logger, errorChan chan<- error) {

	// Reader, instead of Scanner, so we can handle lines > 64k :(
	in := bufio.NewReader(reader)

	for {
		line, prefix, err := in.ReadLine()
		if len(line) > 0 {
			if prefix {
				writer.Printf("%s", line)
			} else {
				writer.Printf("%s\n", line)
			}
		}

		if err != nil {
			if err != io.EOF {
				// We want to send the error,
				// but errorChan could be closed
				select {
				case errorChan <- err:
				default:
				}
			}
			return
		}
	}
}

// Sbuffer is a goro-safe bytes.Buffer
type Sbuffer struct {
	buffer bytes.Buffer
	mutex  sync.Mutex
}

// Write appends the contents of p to the buffer, growing the buffer as needed. It returns
// the number of bytes written or an error.
func (s *Sbuffer) Write(p []byte) (n int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.Write(p)
}

// String returns the contents of the unread portion of the buffer as a string.
func (s *Sbuffer) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.String()
}
