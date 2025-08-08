package logger

import (
	"bufio"
	"io"
	"sync"
)

const (
	// _defaultBufferSize specifies the default size used by Buffer.
	_defaultBufferSize = 256 * 1024 // 256 kB
)

type BufferedWriter struct {
	Out io.Writer

	// Size specifies the maximum amount of data the writer will buffered
	// before flushing.
	//
	// Defaults to 256 kB if unspecified.
	Size int

	// unexported fields for state
	mu          sync.Mutex
	initialized bool // whether initialize() has run
	writer      *bufio.Writer
}

func (s *BufferedWriter) initialize() {
	size := s.Size
	if size == 0 {
		size = _defaultBufferSize
	}

	s.writer = bufio.NewWriterSize(s.Out, size)
	s.initialized = true
}

// Write writes log data into buffer directly, multiple Write calls will be batched,
// and log data will be flushed to disk when the buffer is full or periodically.
func (s *BufferedWriter) Write(bs []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		s.initialize()
	}

	// To avoid partial writes from being flushed, we manually flush the existing buffer if:
	// * The current write doesn't fit into the buffer fully, and
	// * The buffer is not empty (since bufio will not split large writes when the buffer is empty)
	if len(bs) > s.writer.Available() && s.writer.Buffered() > 0 {
		if err := s.writer.Flush(); err != nil {
			return 0, err
		}
	}

	return s.writer.Write(bs)
}

// Flush flushes buffered log data into disk directly.
func (s *BufferedWriter) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return s.writer.Flush()
	}
	return nil
}
