package logger

import (
	"bufio"
	"io"
	"sync"
	"time"
)

const (
	// _defaultBufferSize specifies the default size used by Buffer.
	_defaultBufferSize = 256 * 1024 // 256 kB

	// _defaultFlushInterval specifies the default flush interval for
	// Buffer.
	_defaultFlushInterval = 30 * time.Second
)

type BufferedWriter struct {
	Out io.Writer

	// Size specifies the maximum amount of data the writer will buffered
	// before flushing.
	//
	// Defaults to 256 kB if unspecified.
	Size int

	// FlushInterval specifies how often the writer should flush data if
	// there have been no writes.
	//
	// Defaults to 30 seconds if unspecified.
	FlushInterval time.Duration

	// unexported fields for state
	mu          sync.Mutex
	initialized bool // whether initialize() has run
	stopped     bool // whether Stop() has run
	writer      *bufio.Writer
	ticker      *time.Ticker
	stop        chan struct{} // closed when flushLoop should stop
	done        chan struct{} // closed when flushLoop has stopped
}

func (s *BufferedWriter) initialize() {
	size := s.Size
	if size == 0 {
		size = _defaultBufferSize
	}

	flushInterval := s.FlushInterval
	if flushInterval == 0 {
		flushInterval = _defaultFlushInterval
	}

	s.ticker = time.NewTicker(flushInterval)
	s.writer = bufio.NewWriterSize(s.Out, size)
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	s.initialized = true
	go s.flushLoop()
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

// flushLoop flushes the buffer at the configured interval until Stop is
// called.
func (s *BufferedWriter) flushLoop() {
	defer close(s.done)

	for {
		select {
		case <-s.ticker.C:
			// we just simply ignore error here
			// because the underlying bufio writer stores any errors
			// and we return any error from Flush() as part of the close
			_ = s.Flush()
		case <-s.stop:
			return
		}
	}
}
