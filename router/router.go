package router

import (
	"fmt"
	"io/ioutil"
	"os"

	kv "gopkg.in/Clever/kayvee-go.v5"
	"gopkg.in/yaml.v2"
)

var (
	appName  string
	teamName string
)

func init() {
	appName = os.Getenv("_APP_NAME")
	if appName == "" {
		appName = "UNSET"
	}
	teamName = os.Getenv("_TEAM_OWNER")
	if teamName == "" {
		teamName = "UNSET"
	}
}

// Route returns routing metadata for the log line `msg`. The outputs (with
// variable substitutions performed) for each rule matched are placed under the
// "routes" key.
func (r RuleRouter) Route(msg map[string]interface{}) map[string]interface{} {
	outputs := []map[string]interface{}{}
	for _, rule := range r.rules {
		if rule.Matches(msg) {
			outputs = append(outputs, rule.OutputFor(msg))
		}
	}
	return map[string]interface{}{
		"app":         appName,
		"team":        teamName,
		"kv_version":  kv.Version,
		"kv_language": "go",
		"routes":      outputs,
	}
}

// NewFromConfig constructs a Router using the configuration specified as yaml
// in `filename`. The routing rules should be placed under the "routes" key on
// the root-level map in the file. Validation is performed as described in
// parse.go.
func NewFromConfig(filename string) (Router, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	fileBytes, err := ioutil.ReadAll(file)
	file.Close()
	router, err := newFromConfigBytes(fileBytes)
	if err != nil {
		return nil, fmt.Errorf(
			"Error initializing kayvee log router from file '%s':\n%s",
			filename, err.Error(),
		)
	}
	return router, nil
}

func newFromConfigBytes(fileBytes []byte) (RuleRouter, error) {
	var config struct {
		Routes map[string]Rule
	}
	// Unmarshaling also validates the config
	err := yaml.Unmarshal(fileBytes, &config)
	if err != nil {
		return RuleRouter{}, err
	}
	router := RuleRouter{}
	for name, rule := range config.Routes {
		rule.Name = name
		router.rules = append(router.rules, rule)
	}
	return router, err
}
