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
	InfoD(title string, data map[string]any)
	WarnD(title string, data map[string]any)
	ErrorD(title string, data map[string]any)
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

	rollups sync.Map
}

// NewRollupRouter creates a new log rollup output.
// Rollups will stop when the context is canceled.
func NewRollupRouter(ctx context.Context, logger RollupLogger, reportingDelay time.Duration) *RollupRouter {
	l := &RollupRouter{
		logger:         logger,
		reportingDelay: reportingDelay,
		rollups:        sync.Map{},
		ctx:            ctx,
	}
	go func() {
		<-ctx.Done()
		l.rollups.Clear()
	}()

	return l
}

// ShouldRollup returns true when a log msg meets the criteria for rollup.
// In the future allow more configurability, for now default to 2xx's and < 500ms.
func (r *RollupRouter) ShouldRollup(logmsg map[string]any) bool {
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
func (r *RollupRouter) Process(logmsg map[string]any) {
	select {
	case <-r.ctx.Done():
		return
	default:
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

	r.add(statusCode, op, httpMethod, logmsg)
}

func (r *RollupRouter) add(statusCode int, op, method string, logmsg map[string]any) {
	key := fmt.Sprintf("%d-%s-%s", statusCode, method, op)

	rollup, ok := r.rollups.LoadOrStore(key, &logRollup{
		logger:     r.logger,
		statusCode: statusCode,
		op:         op,
		httpMethod: method,
	})

	ru := rollup.(*logRollup)
	if !ok {
		go ru.report(r.ctx, r.reportingDelay)
	}

	atomic.AddInt64(&ru.count, 1)
	atomic.AddInt64(&ru.rollupResponseTimeNsSum, logmsg["response-time"].(time.Duration).Nanoseconds())
}

func (r *RollupRouter) Flush() {
	r.rollups.Range(func(_, value any) bool {
		rollup := value.(*logRollup)
		rollup.flush()
		return true
	})
}

// logRollup represents a single rollup.
type logRollup struct {
	logger                  RollupLogger
	statusCode              int
	op                      string
	httpMethod              string
	count                   int64
	rollupResponseTimeNsSum int64
}

func (r *logRollup) report(ctx context.Context, interval time.Duration) {
	t := time.Tick(interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t:
			r.flush()
		}
	}
}

func (r *logRollup) flush() {
	count := atomic.SwapInt64(&r.count, 0)
	sum := atomic.SwapInt64(&r.rollupResponseTimeNsSum, 0) / int64(time.Millisecond)

	if count == 0 {
		// no logs yet so return
		return
	}

	msg := logger.M{
		"status-code":          r.statusCode,
		"op":                   r.op,
		"method":               r.httpMethod,
		"via":                  "kayvee-middleware",
		"count":                count,
		"response-time-ms-sum": sum,
		"response-time-ms":     sum / count,
	}

	switch logLevelFromStatus(r.statusCode) {
	case logger.Error:
		r.logger.ErrorD("request-finished-rollup", msg)
	case logger.Warning:
		r.logger.WarnD("request-finished-rollup", msg)
	default:
		r.logger.InfoD("request-finished-rollup", msg)
	}
}
