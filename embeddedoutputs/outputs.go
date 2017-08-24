package embeddedoutputs

import "errors"

// Output is a destination you'd like to send log data to.
type Output interface {
	Process(data, config map[string]interface{}) error
}

var (
	logRollupOutputName   = "log-rollup"
	globalLogRollupOutput = (Output)(nil)
)

func embeddedOutput(outputName string) bool {
	switch outputName {
	case logRollupOutputName:
		return true
	}
	return false
}

// InitLogRollupOutput must be called before routing logs to the log rollup output.
func InitLogRollupOutput(logRollupOutput *LogRollupOutput) {
	globalLogRollupOutput = logRollupOutput
}

// Route a log to an embedded output, if specificed in the routing data in _kvmeta.
func Route(data map[string]interface{}) (bool, error) {
	routed := false
	if kvmeta, ok := data["_kvmeta"].(map[string]interface{}); ok {
		for _, output := range kvmeta["routes"].([]map[string]interface{}) {
			outputType, ok := output["type"].(string)
			if ok && embeddedOutput(outputType) {
				routed = true
				if err := callEmbeddedOutput(outputType, data, output); err != nil {
					return routed, err
				}
			}
		}
	}
	return routed, nil
}

func callEmbeddedOutput(outputType string, data, config map[string]interface{}) error {
	switch outputType {
	case logRollupOutputName:
		if globalLogRollupOutput == nil {
			return errors.New("must call InitLogRollupOutput")
		}
		return globalLogRollupOutput.Process(data, config)
	}
	return nil
}
