package middleware

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type RollupLoggerCall struct {
	Title string
	Data  map[string]interface{}
}

type MockRollupLogger struct {
	mu          sync.Mutex
	calls       []RollupLoggerCall
	infoDCalls  []RollupLoggerCall
	errorDCalls []RollupLoggerCall
}

func (m *MockRollupLogger) InfoD(title string, data map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	call := RollupLoggerCall{title, data}
	m.calls = append(m.calls, call)
	m.infoDCalls = append(m.infoDCalls, call)
}

func (m *MockRollupLogger) ErrorD(title string, data map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	call := RollupLoggerCall{title, data}
	m.calls = append(m.calls, call)
	m.errorDCalls = append(m.errorDCalls, call)
}

func TestRollups(t *testing.T) {
	mockLogger := &MockRollupLogger{}
	reportingDelay := 1 * time.Second
	rr := NewRollupRouter(context.Background(), mockLogger, reportingDelay)

	// send a bunch of data to the rollup router in parallel (since logging can
	// happen from multiple goroutines) and you should see it logged as a rollup
	type logmsg struct {
		StatusCode   int
		Path         string
		Canary       bool
		ResponseTime time.Duration
	}
	for _, msg := range []logmsg{
		{
			StatusCode:   500,
			Path:         "/servererror",
			Canary:       false,
			ResponseTime: 1000 * time.Millisecond,
		},
		{
			StatusCode:   404,
			Path:         "/notfound",
			Canary:       false,
			ResponseTime: 10 * time.Millisecond,
		},
		{
			StatusCode:   200,
			Path:         "/healthcheck",
			Canary:       false,
			ResponseTime: 100 * time.Millisecond,
		},
	} {
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			go func(msg logmsg) {
				wg.Add(1)
				defer wg.Done()
				rr.Process(map[string]interface{}{
					"status-code":   msg.StatusCode,
					"path":          msg.Path,
					"canary":        msg.Canary,
					"response-time": msg.ResponseTime,
				})
			}(msg)
		}
		wg.Wait()
	}
	time.Sleep(reportingDelay + 100*time.Millisecond) // check after reporting delay

	assert.Equal(t, mockLogger.errorDCalls, []RollupLoggerCall{
		{
			Title: "request-finished-rollup",
			Data: map[string]interface{}{
				"canary":            false,
				"count":             int64(100),
				"path":              "/servererror",
				"response-time":     int64(1000000000),
				"response-time-sum": int64(100000000000),
				"status-code":       500,
				"via":               "kayvee-middleware",
			},
		},
	})
	assert.Equal(t, mockLogger.infoDCalls, []RollupLoggerCall{
		{
			Title: "request-finished-rollup",
			Data: map[string]interface{}{
				"canary":            false,
				"count":             int64(100),
				"path":              "/notfound",
				"response-time":     int64(10000000),
				"response-time-sum": int64(1000000000),
				"status-code":       404,
				"via":               "kayvee-middleware",
			},
		},
		{
			Title: "request-finished-rollup",
			Data: map[string]interface{}{
				"canary":            false,
				"count":             int64(100),
				"path":              "/healthcheck",
				"response-time":     int64(100000000),
				"response-time-sum": int64(10000000000),
				"status-code":       200,
				"via":               "kayvee-middleware",
			},
		},
	})
}
