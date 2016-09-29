package router

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Clever/go-utils/stringset"
)

// UnmarshalYAML unmarshals the `matchers` section of a log-routing
// configuration. It ensures that this section is a map from strings to either
// a single string or a list of strings and that:
// - Keys and values contain no special characters -- $, %, {, or } -- and
//   hence no substitutions
// - If value is a list, there are no repeated elements
func (m *RuleMatchers) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var data map[string]interface{}

	err := unmarshal(&data)
	if err != nil {
		return err
	}

	*m = RuleMatchers(make(map[string][]string))

	for field, value := range data {
		err := ensureNoSpecials(field)
		if err != nil {
			return fmt.Errorf("Invalid matcher field name '%s': %s", field, err.Error())
		}

		typeErr := fmt.Errorf("Invalid type for matcher value on field '%s': %+v.\n\tShould be string or slice of strings.", field, value)
		switch value := value.(type) {
		case string:
			err := ensureNoSpecials(value)
			if err != nil {
				return fmt.Errorf("Invalid matcher value '%s' on field '%s': %s",
					value, field, err.Error())
			}
			(*m)[field] = []string{value}
		case []interface{}:
			strValue := make([]string, len(value))
			for i, v := range value {
				if str, ok := v.(string); ok {
					err := ensureNoSpecials(str)
					if err != nil {
						return fmt.Errorf("Invalid matcher value '%s' on field '%s': %s",
							str, field, err.Error())
					}
					strValue[i] = str
				} else {
					return typeErr
				}
			}
			err := ensureNoRepeats(strValue)
			if err != nil {
				return fmt.Errorf("Invalid matcher value on field '%s': %s", field, err.Error())
			}
			(*m)[field] = strValue
		default:
			return typeErr
		}
	}

	return nil
}

// UnmarshalYAML unmarshals the `output` section of a log-routing
// configuration. It ultimately just unmarshals into a map[string]interface{}
// but before doing so it uses the WhateverOutput structs defined in rule.go to
// validate the format are correct for whatever type of output is specified.
func (o *RuleOutput) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var obj struct {
		Type string
	}
	err := unmarshal(&obj)
	if err != nil {
		return err
	}

	switch obj.Type {
	case "metrics":
		var metrics metricsOutput
		err := validateStrict(&metrics, unmarshal)
		if err != nil {
			return err
		}
	case "alert":
		var alert alertOutput
		err := validateStrict(&alert, unmarshal)
		if err != nil {
			return err
		}
	case "analytics":
		var analytics analyticsOutput
		err := validateStrict(&analytics, unmarshal)
		if err != nil {
			return err
		}
	case "notification":
		var notification notificationOutput
		err := validateStrict(&notification, unmarshal)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown output type: %s", obj.Type)
	}

	var result map[string]interface{}
	err = unmarshal(&result)
	*o = result
	return err
}

// ensureNoSpecials returns an error iff s contains a special character
func ensureNoSpecials(s string) error {
	specials := []rune{'$', '%', '{', '}'}
	for _, r := range specials {
		if strings.ContainsRune(s, r) {
			return fmt.Errorf("Contains illegal character: '%c'\n"+
				"\tThe characters '$', '%%', '{', and '}' aren't allowed here.", r)
		}
	}
	return nil
}

// ensureNoRepeats returns an error iff list contains the same string two or
// more times
func ensureNoRepeats(list []string) error {
	s := stringset.FromList(list)
	if len(s) < len(list) {
		return fmt.Errorf("List %v contains repeated elements.", list)
	}
	return nil
}

// yamlName returns the name of the provided struct field that the yaml parser
// will use. This is what is specified in the struct tag if provided, or just
// the field name lowercased.
func yamlName(f reflect.StructField) string {
	tagElems := strings.Split(f.Tag.Get("yaml"), ",")
	if len(tagElems) > 0 && tagElems[0] != "" {
		return tagElems[0]
	}
	return strings.ToLower(f.Name)
}

// validateStrict ensures that the following is true of the yaml behind the
// `unmarshal` function:
// - The types are set in accordance with format
// - All keys are explicitly set
// - No extra keys are set
func validateStrict(format interface{}, unmarshal func(interface{}) error) error {
	// Validate types are correct
	err := unmarshal(format)
	if err != nil {
		return err
	}

	var raw struct {
		Type string
		Data map[string]interface{} `yaml:",inline"`
	}
	err = unmarshal(&raw)
	if err != nil {
		return err
	}

	// Validate all fields explicitly set
	ft := reflect.TypeOf(format).Elem()
	for i := 0; i < ft.NumField(); i++ {
		field := ft.Field(i)
		name := yamlName(field)
		if _, ok := raw.Data[name]; !ok {
			return fmt.Errorf("Output type '%s' missing key: %s", raw.Type, name)
		}
		delete(raw.Data, name)
	}

	// Validate no extra fields set
	if len(raw.Data) != 0 {
		return fmt.Errorf("Output type '%s' contains unknown keys: %+v", raw.Type, raw.Data)
	}

	return nil
}
