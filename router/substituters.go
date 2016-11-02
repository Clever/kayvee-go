package router

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var envvarTokens = regexp.MustCompile(`\$\{.+?\}`)
var fieldTokens = regexp.MustCompile(`%\{.+?\}`)

// substitute performs a key-value substitution on `obj`, replacing instances
// of tokenMatcher in the keys or values of `obj` with `replacer(name)`. It
// returns the substituted map and does not modify the input map.
func substitute(
	obj map[string]interface{}, tokenMatcher *regexp.Regexp, replacer func(key string) string,
) map[string]interface{} {
	newObj := make(map[string]interface{})
	for k, v := range obj {
		switch v := v.(type) {
		case string:
			newObj[k] = tokenMatcher.ReplaceAllStringFunc(v, replacer)
		case []string:
			newV := make([]string, len(v))
			for i := 0; i < len(v); i++ {
				newV[i] = tokenMatcher.ReplaceAllStringFunc(v[i], replacer)
			}
			newObj[k] = newV
		default:
			newObj[k] = v
		}
	}
	return newObj
}

func substituteEnvVars(data map[string]interface{}) (map[string]interface{}, error) {
	envErrors := []string{}
	getEnvOrErr := func(key string) string {
		// Performance optimization: slice sub-sequence is faster than regex.FindStringSubmatch
		key = key[2 : len(key)-1]

		val := os.Getenv(key)
		if val == "" {
			envErrors = append(envErrors, fmt.Sprintf("\tEnvironment variable '%s' not set", key))
		}
		return val
	}

	subs := substitute(data, envvarTokens, getEnvOrErr)
	if len(envErrors) > 0 {
		return nil, errors.New(strings.Join(envErrors, "\n"))
	}

	return subs, nil
}

func substituteFields(
	data map[string]interface{}, lookup func(string) (string, bool),
) map[string]interface{} {
	kvSubber := func(key string) string {
		// Performance optimization: slice sub-sequence is faster than regex.FindStringSubmatch
		key = key[2 : len(key)-1]

		if v, ok := lookup(key); ok {
			return v
		}
		return "KEY_NOT_FOUND"
	}

	return substitute(data, fieldTokens, kvSubber)
}
