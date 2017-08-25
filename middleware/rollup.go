package middleware

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/Clever/kayvee-go.v6/logger"
)

var globalRollupRouter *RollupRouter

// RollupLogger will log info / error rollups depending on status code.
type RollupLogger interface {
	InfoD(title string, data map[string]interface{})
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

	// create a rollup object per unique (status-code, path) pair
	rollupsMu sync.Mutex
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
		select {
		case <-ctx.Done():
			l.rollupsMu.Lock()
			l.rollups = map[string]*logRollup{}
			l.ctxDone = true
			l.rollupsMu.Unlock()
		}
	}()
	return l
}

// Process a log message to roll up.
func (r *RollupRouter) Process(logmsg map[string]interface{}) {
	if r.ctxDone {
		return
	}

	statusCode, ok := logmsg["status-code"].(int)
	if !ok {
		return
	}
	path, ok := logmsg["path"].(string)
	if !ok {
		return
	}
	canary, ok := logmsg["canary"].(bool)
	if !ok {
		return
	}
	r.findOrCreate(statusCode, path, canary).add(logmsg)
}

func (r *RollupRouter) findOrCreate(statusCode int, path string, canary bool) *logRollup {
	r.rollupsMu.Lock()
	defer r.rollupsMu.Unlock()
	rollupKey := fmt.Sprintf("%d-%s", statusCode, path)
	if canary {
		rollupKey += "-canary"
	}
	if rollup, ok := r.rollups[rollupKey]; ok {
		return rollup
	}
	rollup := &logRollup{
		Logger:           r.logger,
		ReportingDelayNs: (r.reportingDelay).Nanoseconds(),
		StatusCode:       statusCode,
		Path:             path,
		Canary:           canary,
	}
	r.rollups[rollupKey] = rollup
	go rollup.schedule(r.ctx)
	return rollup
}

// logRollup represents a single rollup.
type logRollup struct {
	Logger           RollupLogger
	ReportingDelayNs int64
	StatusCode       int
	Path             string
	Canary           bool

	rollupMu                sync.Mutex
	rollupMsg               map[string]interface{}
	rollupResponseTimeNsSum int64
}

func (r *logRollup) report() {
	r.rollupMu.Lock()
	defer r.rollupMu.Unlock()
	if r.rollupMsg != nil {
		r.rollupMsg["response-time-sum"] = r.rollupResponseTimeNsSum
		r.rollupMsg["response-time"] = r.rollupResponseTimeNsSum / r.rollupMsg["count"].(int64)
		switch logLevelFromStatus(r.StatusCode) {
		case logger.Error:
			r.Logger.ErrorD("request-finished-rollup", r.rollupMsg)
		default:
			r.Logger.InfoD("request-finished-rollup", r.rollupMsg)
		}
		r.rollupMsg = nil
		r.rollupResponseTimeNsSum = 0
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

func (r *logRollup) add(logmsg map[string]interface{}) {
	r.rollupMu.Lock()
	defer r.rollupMu.Unlock()

	if r.rollupMsg == nil {
		r.rollupMsg = map[string]interface{}{
			"status-code": r.StatusCode,
			"path":        r.Path,
			"count":       int64(0),
			"canary":      r.Canary,
			"via":         "kayvee-middleware",
		}
	}

	r.rollupMsg["count"] = r.rollupMsg["count"].(int64) + 1
	r.rollupResponseTimeNsSum += logmsg["response-time"].(time.Duration).Nanoseconds()
}