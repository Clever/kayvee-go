package router

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Matches returns true if the `msg` matches the matchers specified in this
// routing rule.
func (r *Rule) Matches(msg map[string]interface{}) bool {
	for field, values := range r.Matchers {
		if !fieldMatches(field, values, msg) {
			return false
		}
	}
	return true
}

// OutputFor returns the output map for this routing rule with substitutions
// applied in accordance with the current environment and the contents of the
// message.
func (r *Rule) OutputFor(msg map[string]interface{}) map[string]interface{} {
	kvSubber := func(key string) string {
		if v, ok := msg[key]; ok {
			if str, ok := v.(string); ok {
				return str
			}
			return "INVALID_VALUE"
		}
		return "KEY_NOT_FOUND"
	}

	kvSubbed := substitute(r.Output, "%", kvSubber)
	finalSubbed := substitute(kvSubbed, `\$`, os.Getenv)
	finalSubbed["rule"] = r.Name
	return finalSubbed
}

// fieldMatches returns true if the value of the key `field` in the map `obj`
// is one of `values`. Dots in `field` are interpreted as denoting subobjects
// -- i.e. the field name "x.y.z" says to check obj["x"]["y"]["z"].
func fieldMatches(field string, values []string, obj map[string]interface{}) bool {
	if field == "" {
		panic(fmt.Errorf("Invalid field specified"))
	}

	if strings.ContainsRune(field, '.') {
		fieldPath := strings.Split(field, ".")
		if subObj, ok := obj[fieldPath[0]].(map[string]interface{}); ok {
			return fieldMatches(strings.Join(fieldPath[1:], "."), values, subObj)
		}
		// No subobject
		return false
	}

	objVal, ok := obj[field]
	if !ok {
		return false
	}
	for _, v := range values {
		if objVal == v {
			return true
		}
	}
	return false
}

// substitute performs a key-value substitution on `obj`, replacing instances
// of "X{name}" (where "X" is `substKey`) in the keys or values of `obj` with
// `subber(name)`. It returns the substituted map and does not modify the input
// map.
func substitute(obj map[string]interface{}, substKey string, subber func(key string) string) map[string]interface{} {
	newObj := make(map[string]interface{})
	for k, v := range obj {
		switch v := v.(type) {
		case string:
			newObj[k] = doSubstitute(v, substKey, subber)
		case []string:
			newV := make([]string, len(v))
			for i, elem := range v {
				newV[i] = doSubstitute(elem, substKey, subber)
			}
			newObj[k] = newV
		default:
			newObj[k] = v
		}
	}
	return newObj
}

// doSubstitute performs the sort of substitution described in `substitute` on
// the string `str`.
func doSubstitute(str string, substKey string, subber func(key string) string) string {
	re := regexp.MustCompile(substKey + `\{(.*?)\}`)
	repl := func(key string) string {
		name := re.FindStringSubmatch(key)[1]
		return subber(name)
	}
	return re.ReplaceAllStringFunc(str, repl)
}
