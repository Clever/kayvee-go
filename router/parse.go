package router

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v2"
)

func parse(fileBytes []byte) (map[string]Rule, error) {
	var config struct {
		Routes map[string]Rule `json:"routes"`
	}
	// Unmarshaling also validates the config
	err := yaml.Unmarshal(fileBytes, &config)
	if err != nil {
		return config.Routes, err
	}

	schemaLoader := gojsonschema.NewStringLoader(routerSchema)
	docLoader := gojsonschema.NewGoLoader(&config)

	err = validate(schemaLoader, docLoader)
	if err != nil {
		return config.Routes, err
	}

	return config.Routes, nil
}

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
	var rawData map[string][]interface{}
	err := unmarshal(&rawData)
	if err != nil {
		return err
	}

	for key, arr := range rawData {
		for _, val := range arr {
			switch val.(type) {
			case string:
			default:
				return fmt.Errorf(`Invalid log-router matcher -- key: "%s", value: %+#v.  `+
					"Only strings can be matched.", key, val)
			}
		}
	}

	// Now actually do the unmarshaling into the correct type and save it to `m`.
	var data map[string][]string
	err = unmarshal(&data)
	if err != nil {
		return err
	}

	for field, vals := range data {
		for _, val := range vals {
			if val == "*" && len(vals) > 1 {
				return fmt.Errorf("Invalid matcher values in %s.\n"+
					"Wildcard matcher can't co-exist with other matchers.", field)
			}
		}
	}

	*m = data
	return nil
}
