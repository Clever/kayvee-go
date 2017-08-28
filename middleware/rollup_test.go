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
	infoDCalls  []RollupLoggerCall
	errorDCalls []RollupLoggerCall
}

func (m *MockRollupLogger) InfoD(title string, data map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	call := RollupLoggerCall{title, data}
	m.infoDCalls = append(m.infoDCalls, call)
}

func (m *MockRollupLogger) InfoDCalls() []RollupLoggerCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	var calls []RollupLoggerCall
	for _, call := range m.infoDCalls {
		calls = append(calls, call)
	}
	return calls
}

func (m *MockRollupLogger) ErrorD(title string, data map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	call := RollupLoggerCall{title, data}
	m.errorDCalls = append(m.errorDCalls, call)
}

func (m *MockRollupLogger) ErrorDCalls() []RollupLoggerCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	var calls []RollupLoggerCall
	for _, call := range m.errorDCalls {
		calls = append(calls, call)
	}
	return calls
}

func TestProcess(t *testing.T) {
	mockLogger := &MockRollupLogger{}
	reportingDelay := 1 * time.Second
	rr := NewRollupRouter(context.Background(), mockLogger, reportingDelay)

	// send a bunch of data to the rollup router in parallel (since logging can
	// happen from multiple goroutines) and you should see it logged as a rollup
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rr.Process(map[string]interface{}{
				"status-code":   200,
				"path":          "/healthcheck",
				"canary":        false,
				"response-time": 100 * time.Millisecond,
			})
		}()
	}
	wg.Wait()

	// check in shortly after reporting delay
	time.Sleep(reportingDelay + 500*time.Millisecond)

	assert.Equal(t, mockLogger.InfoDCalls(), []RollupLoggerCall{
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

func TestRollup(t *testing.T) {
	mockLogger := &MockRollupLogger{}
	reportingDelay := 1 * time.Second
	rr := NewRollupRouter(context.Background(), mockLogger, reportingDelay)

	// if a request is a 200 or is too slow, it should not get rolled up
	for _, falseyInput := range []map[string]interface{}{
		map[string]interface{}{
			"status-code":   200,
			"path":          "/",
			"canary":        true,
			"response-time": 600 * time.Millisecond, // too slow
		},
		map[string]interface{}{
			"status-code":   500, // not a 200
			"path":          "/",
			"canary":        true,
			"response-time": 100 * time.Millisecond,
		},
	} {
		assert.Equal(t, rr.Rollup(falseyInput), false, "expected false return: %v", falseyInput)
	}

	// 200s that are fast enough should get rolled up
	for _, truthyInput := range []map[string]interface{}{
		map[string]interface{}{
			"status-code":   200,
			"path":          "/bar",
			"canary":        true,
			"response-time": 100 * time.Millisecond,
		},
		map[string]interface{}{
			"status-code":   200,
			"path":          "/foo",
			"canary":        true,
			"response-time": 400 * time.Millisecond,
		},
	} {
		assert.Equal(t, rr.Rollup(truthyInput), true, "expected true return: %v", truthyInput)
	}
}
