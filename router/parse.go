package router

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

func validate(schemaLoader, docLoader gojsonschema.JSONLoader) error {
	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return err
	}
	if !result.Valid() {
		errStrings := make([]string, len(result.Errors()))
		for idx, err := range result.Errors() {
			errStrings[idx] = fmt.Sprintf("\t%s: %s", err, err.Value())
		}
		return errors.New(strings.Join(errStrings, "\n"))
	}
	return nil
}

// UnmarshalYAML unmarshals the `matchers` section of a log-routing
// configuration and validates it.
func (m *RuleMatchers) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Use a map[string]interface{} for validation purposes. If we used a
	// map[string][]string, the YAML unmarshaler would coerce non-string values
	// into string values, breaking our ability to validate configs. i.e., it
	// would change `title: [7, []]` into `title: ["7", "[]"]`. Using a
	// map[string]interface{} tells the unmarshaler to use natural types.
	var rawData map[string]interface{}
	err := unmarshal(&rawData)
	if err != nil {
		return err
	}

	schemaLoader := gojsonschema.NewStringLoader(matchersSchema)
	docLoader := gojsonschema.NewGoLoader(&rawData)
	err = validate(schemaLoader, docLoader)
	if err != nil {
		return err
	}

	// Now actually do the unmarshaling into the correct type and save it to
	// `m`.
	var data map[string][]string
	err = unmarshal(&data)
	if err != nil {
		return err
	}
	*m = data
	return nil
}

// UnmarshalYAML unmarshals the `output` section of a log-routing
// configuration. It also validates against the schema.
func (o *RuleOutput) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var rawData map[string]interface{}
	err := unmarshal(&rawData)
	if err != nil {
		return err
	}
	outputType, ok := rawData["type"].(string)
	if !ok {
		return fmt.Errorf("Output missing type")
	}

	var schema string
	switch outputType {
	case "metrics":
		schema = metricsSchema
	case "alerts":
		schema = alertSchema
	case "analytics":
		schema = analyticsSchema
	case "notifications":
		schema = notificationSchema
	default:
		return fmt.Errorf("\tOuput type not valid: %s", outputType)
	}
	schemaLoader := gojsonschema.NewStringLoader(schema)
	docLoader := gojsonschema.NewGoLoader(&rawData)
	err = validate(schemaLoader, docLoader)
	if err != nil {
		return err
	}

	envErrors := []string{}
	getEnvOrErr := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			envErrors = append(envErrors, fmt.Sprintf("\tEnvironment variable '%s' not set", key))
		}
		return val
	}
	*o = substitute(rawData, `\$`, getEnvOrErr)
	if len(envErrors) > 0 {
		return errors.New(strings.Join(envErrors, "\n"))
	}

	return nil
}
