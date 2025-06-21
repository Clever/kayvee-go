package middleware

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Clever/kayvee-go/v7/logger"
)

var globalRollupRouter *RollupRouter

// RollupLogger will log info / error rollups depending on status code.
type RollupLogger interface {
	InfoD(title string, data map[string]interface{})
	WarnD(title string, data map[string]interface{})
	ErrorD(title string, data map[string]interface{})
}

// EnableRollups turns on rollups for kv middleware logs.
func EnableRollups(ctx context.Context, logger RollupLogger, reportingInterval time.Duration) {
	globalRollupRouter = NewRollupRouter(ctx, logger, reportingInterval)
}

// RollupRouter rolls up log lines and periodically logs them as one log line.
type RollupRouter struct {
	logger         RollupLogger
	reportingDelay time.Duration
	ctx            context.Context
	ctxDone        bool

	// create a rollup object per unique (status-code, op) pair
	rollupsMu sync.RWMutex
	rollups   map[string]*logRollup
}

// NewRollupRouter creates a new log rollup output.
// Rollups will stop when the context is canceled.
func NewRollupRouter(ctx context.Context, logger RollupLogger, reportingDelay time.Duration) *RollupRouter {
	l := &RollupRouter{
		logger:         logger,
		reportingDelay: reportingDelay,
		rollups:        map[string]*logRollup{},
		ctx:            ctx,
		ctxDone:        false,
	}
	go func() {
		<-ctx.Done()
		l.rollupsMu.Lock()
		l.rollups = map[string]*logRollup{}
		l.ctxDone = true
		l.rollupsMu.Unlock()
	}()
	return l
}

// ShouldRollup returns true when a log msg meets the criteria for rollup.
// In the future allow more configurability, for now default to 2xx's and < 500ms.
func (r *RollupRouter) ShouldRollup(logmsg map[string]interface{}) bool {
	if _, ok := logmsg["op"].(string); !ok {
		return false
	}

	if _, ok := logmsg["method"].(string); !ok {
		return false
	}

	statusCode, ok := logmsg["status-code"].(int)
	if !ok {
		return false
	} else if !(statusCode >= 200 && statusCode < 300) {
		return false
	}

	responseTime, ok := logmsg["response-time"].(time.Duration)
	if !ok {
		return false
	} else if responseTime >= 500*time.Millisecond {
		return false
	}

	return true
}

// Process rolls up a log message.
func (r *RollupRouter) Process(logmsg map[string]interface{}) {
	if r.ctxDone {
		return
	}

	statusCode, ok := logmsg["status-code"].(int)
	if !ok {
		return
	}
	op, ok := logmsg["op"].(string)
	if !ok {
		return
	}
	httpMethod, ok := logmsg["method"].(string)
	if !ok {
		return
	}
	r.findOrCreate(statusCode, op, httpMethod).add(logmsg)
}

func (r *RollupRouter) findOrCreate(statusCode int, op, method string) *logRollup {
	rollupKey := fmt.Sprintf("%d-%s-%s", statusCode, method, op)

	r.rollupsMu.RLock()
	if rollup, ok := r.rollups[rollupKey]; ok {
		r.rollupsMu.RUnlock()
		return rollup
	}
	r.rollupsMu.RUnlock()

	r.rollupsMu.Lock()
	rollup := &logRollup{
		Logger:           r.logger,
		ReportingDelayNs: (r.reportingDelay).Nanoseconds(),
		StatusCode:       statusCode,
		Op:               op,
		HTTPMethod:       method,
	}
	r.rollups[rollupKey] = rollup
	r.rollupsMu.Unlock()

	go rollup.schedule(r.ctx)
	return rollup
}

// logRollup represents a single rollup.
type logRollup struct {
	Logger           RollupLogger
	ReportingDelayNs int64
	StatusCode       int
	Op               string
	HTTPMethod       string

	rollupMu                sync.Mutex
	rollupMsg               map[string]interface{}
	count                   int64
	rollupResponseTimeNsSum int64
}

func (r *logRollup) report() {
	r.rollupMu.Lock()
	defer r.rollupMu.Unlock()
	if r.rollupMsg != nil {
		count := atomic.LoadInt64(&r.count)
		sum := atomic.LoadInt64(&r.rollupResponseTimeNsSum) / int64(time.Millisecond)
		r.rollupMsg["count"] = count
		r.rollupMsg["response-time-ms-sum"] = sum
		r.rollupMsg["response-time-ms"] = sum / count

		switch logLevelFromStatus(r.StatusCode) {
		case logger.Error:
			r.Logger.ErrorD("request-finished-rollup", r.rollupMsg)
		case logger.Warning:
			r.Logger.WarnD("request-finished-rollup", r.rollupMsg)
		default:
			r.Logger.InfoD("request-finished-rollup", r.rollupMsg)
		}
		r.rollupMsg = nil
		atomic.StoreInt64(&r.count, 0)
		atomic.StoreInt64(&r.rollupResponseTimeNsSum, 0)
	}
}

func (r *logRollup) schedule(ctx context.Context) {
	lastReport := time.Now()
	for {
		reportingDelay := time.Duration(atomic.LoadInt64(&r.ReportingDelayNs))
		wakeupTime := lastReport.Add(reportingDelay)
		now := time.Now()
		if now.After(wakeupTime) {
			wakeupTime = now.Add(reportingDelay)
		}
		sleepTime := wakeupTime.Sub(now)

		select {
		case <-ctx.Done():
			return
		case <-time.After(sleepTime):
			lastReport = time.Now()
			r.report()
		}
	}
}

func (r *logRollup) add(logmsg map[string]any) {
	atomic.AddInt64(&r.count, 1)
	atomic.AddInt64(&r.rollupResponseTimeNsSum, logmsg["response-time"].(time.Duration).Nanoseconds())

	r.rollupMu.Lock()
	defer r.rollupMu.Unlock()
	if r.rollupMsg == nil {
		r.rollupMsg = map[string]any{
			"status-code": r.StatusCode,
			"op":          r.Op,
			"method":      r.HTTPMethod,
			"via":         "kayvee-middleware",
		}
	}
}
