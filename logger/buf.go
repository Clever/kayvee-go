package logger

import (
	"bufio"
	"io"
	"log"
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

type bufferedWriter struct {
	out io.Writer

	mu          sync.Mutex
	initialized bool // whether initialize() has run
	stopped     bool // whether Stop() has run
	writer      *bufio.Writer
	ticker      *time.Ticker
	chStop      chan struct{} // closed when flushLoop should stop
	done        chan struct{} // closed when flushLoop has stopped
}

func (s *bufferedWriter) initialize() {
	s.ticker = time.NewTicker(_defaultFlushInterval)
	s.writer = bufio.NewWriterSize(s.out, _defaultBufferSize)
	s.chStop = make(chan struct{})
	s.done = make(chan struct{})
	s.initialized = true
	go s.flushLoop()
}

// Write writes log data into the buffer directly, multiple Write calls
// will be batched, and log data will be flushed to disk when the buffer
// is full or periodically.
func (s *bufferedWriter) Write(bs []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		s.initialize()
	}

	// To avoid partial writes from being flushed, we manually flush the
	// existing buffer if: * The current write doesn't fit into the
	// buffer fully, and * The buffer is not empty (since bufio will not
	// split large writes when the buffer is empty)
	if len(bs) > s.writer.Available() && s.writer.Buffered() > 0 {
		if err := s.writer.Flush(); err != nil {
			return 0, err
		}
	}

	return s.writer.Write(bs)
}

// flush flushes buffered log data into disk directly.
func (s *bufferedWriter) flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	if s.initialized {
		err = s.writer.Flush()
	}
	if err != nil {
		log.Printf("ERROR: Failed to flush log buffer: %v\n", err)
	}
	return err
}

// flushLoop flushes the buffer at the configured interval until Stop is
// called.
func (s *bufferedWriter) flushLoop() {
	defer close(s.done)

	for {
		select {
		case <-s.ticker.C:
			// we just simply ignore error here because flush already
			// logs straight to stderr
			_ = s.flush()
		case <-s.chStop:
			return
		}
	}
}

// stop closes the buffer, cleans up background goroutines, and flushes
// remaining unwritten data.
func (s *bufferedWriter) stop() (err error) {
	// Critical section.
	stopped := func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()

		if !s.initialized {
			return false
		}

		if s.stopped {
			return false
		}
		s.stopped = true

		s.ticker.Stop()
		close(s.chStop) // tell flushLoop to stop
		return true
	}()

	// Not initialized, or already stopped, no need for any cleanup.
	if !stopped {
		return
	}

	// Wait for flushLoop to end outside of the lock, as it may need the lock to complete.
	// See https://github.com/uber-go/zap/issues/1428 for details.
	<-s.done

	return s.flush()
}
