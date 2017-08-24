package embeddedoutputs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// InfoD is what the rollup uses to output rollup info.
type InfoD interface {
	InfoD(title string, data map[string]interface{})
}

// InfoDCreator is a function that returns an InfoD.
// Every rollup is configured with a source + title string, and InfoD will log them.
type InfoDCreator func(source string) InfoD

// LogRollupOutput rolls up log lines and periodically logs them as one log line.
type LogRollupOutput struct {
	infoDCreator InfoDCreator
	ctx          context.Context

	rollupsMu sync.Mutex
	rollups   map[string]*logRollup
}

// NewLogRollupOutput creates a new log rollup output.
func NewLogRollupOutput(infoDCreator InfoDCreator) *LogRollupOutput {
	return &LogRollupOutput{
		infoDCreator: infoDCreator,
		rollups:      map[string]*logRollup{},
		ctx:          context.Background(),
	}
}

// WithContext adds a context to the log rollup output. All rollups will stop reporting
// when the context is canceled.
func (r *LogRollupOutput) WithContext(ctx context.Context) *LogRollupOutput {
	r.ctx = ctx
	return r
}

// Process a message rollup.
func (r *LogRollupOutput) Process(data, config map[string]interface{}) error {
	rollup, err := r.findOrCreate(data, config)
	if err != nil {
		return err
	}
	rollup.add(data)
	return nil
}

type logRollupConfig struct {
	Name       string
	Source     string
	Title      string
	Dimensions []string
}

func parseLogRollupConfig(config map[string]interface{}) (*logRollupConfig, error) {
	var name, source, title string
	var dimensions []string

	if nameI, ok := config["rule"]; !ok {
		return nil, errors.New("rollup doesn't have name")
	} else if nameString, ok := nameI.(string); !ok {
		return nil, errors.New("rollup name is not a string")
	} else {
		name = nameString
	}

	if sourceI, ok := config["source"]; !ok {
		return nil, errors.New("rollup doesn't have source")
	} else if sourceString, ok := sourceI.(string); !ok {
		return nil, errors.New("rollup source is not a string")
	} else {
		source = sourceString
	}

	if titleI, ok := config["title"]; !ok {
		return nil, errors.New("rollup doesn't have title")
	} else if titleString, ok := titleI.(string); !ok {
		return nil, errors.New("rollup title is not a string")
	} else {
		title = titleString
	}

	if dimensionsI, ok := config["dimensions"]; !ok {
		return nil, errors.New("rollup doesn't have dimensions")
	} else if dimensionsArray, ok := dimensionsI.([]interface{}); !ok {
		return nil, errors.New("rollup dimensions is not an interface array")
	} else {
		for _, dimI := range dimensionsArray {
			dimensions = append(dimensions, dimI.(string))
		}
	}

	return &logRollupConfig{
		Name:       name,
		Source:     source,
		Title:      title,
		Dimensions: dimensions,
	}, nil
}

func (r *LogRollupOutput) findOrCreate(data, config map[string]interface{}) (*logRollup, error) {
	logRollupConfig, err := parseLogRollupConfig(config)
	if err != nil {
		return nil, err
	}

	rollupKey := logRollupConfig.Name
	for _, dim := range logRollupConfig.Dimensions {
		dimV, ok := data[dim]
		if !ok {
			return nil, fmt.Errorf("could not find dimension %s in log data", dim)
		}
		rollupKey += fmt.Sprintf("-%s", dimV)
	}

	r.rollupsMu.Lock()
	defer r.rollupsMu.Unlock()
	if rollup, ok := r.rollups[rollupKey]; ok {
		return rollup, nil
	}
	rollup := &logRollup{
		Logger:           r.infoDCreator(logRollupConfig.Source),
		Name:             logRollupConfig.Name,
		Title:            logRollupConfig.Title,
		Dimensions:       logRollupConfig.Dimensions,
		ReportingDelayNs: (time.Second * 20).Nanoseconds(), // todo: make configurable
	}
	r.rollups[rollupKey] = rollup
	go rollup.schedule(r.ctx)
	return rollup, nil
}

// logRollup represents a single rollup.
type logRollup struct {
	Logger           InfoD
	Name             string
	Title            string
	Dimensions       []string
	ReportingDelayNs int64

	rollupMsgMu sync.Mutex
	rollupMsg   map[string]interface{}
}

func (r *logRollup) report() {
	r.rollupMsgMu.Lock()
	defer r.rollupMsgMu.Unlock()
	if r.rollupMsg != nil {
		r.Logger.InfoD(r.Name, r.rollupMsg)
	}
	r.rollupMsg = nil
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

func (r *logRollup) add(msg map[string]interface{}) {
	r.rollupMsgMu.Lock()
	defer r.rollupMsgMu.Unlock()

	if r.rollupMsg == nil {
		r.rollupMsg = map[string]interface{}{
			"count": int64(0),
		}
	}

	r.rollupMsg["count"] = r.rollupMsg["count"].(int64) + 1
	// TODO: allow users to perform additional numeric rollups, e.g. if there's a field `response-time`, let them average it

	for _, dim := range r.Dimensions {
		r.rollupMsg[dim] = msg[dim]
	}
}
