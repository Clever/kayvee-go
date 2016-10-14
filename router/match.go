package router

import (
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
		if v, ok := lookupString(key, msg); ok {
			return v
		}
		return "KEY_NOT_FOUND"
	}

	subbed := substitute(r.Output, "%", kvSubber)
	subbed["rule"] = r.Name
	return subbed
}

// lookupString does an extended lookup on `obj`, interpreting dots in field as
// corresponding to subobjects. It returns the value and true if the lookup
// succeeded or `"", false` if the key is missing or corresponds to a
// non-string value.
func lookupString(field string, obj map[string]interface{}) (string, bool) {
	path := strings.Split(field, ".")
	if len(path) == 0 {
		// `field` is the empty string
		val, ok := obj[field].(string)
		return val, ok
	}
	return lookupStringPath(path, obj)
}

// lookupStringPath does an extended lookup on `obj`, with each entry in `fieldPath`
// corresponding to subobjects. It returns the value and true if the lookup
// succeeded or `"", false` if a key was missing along the path or if the final
// key corresponds to a non-string value.
func lookupStringPath(fieldPath []string, obj map[string]interface{}) (string, bool) {
	if len(fieldPath) == 1 {
		val, ok := obj[fieldPath[0]].(string)
		return val, ok
	}
	if subObj, ok := obj[fieldPath[0]].(map[string]interface{}); ok {
		return lookupStringPath(fieldPath[1:], subObj)
	}
	return "", false
}

// fieldMatches returns true if the value of the key `field` in the map `obj`
// is one of `values`. Dots in `field` are interpreted as denoting subobjects
// -- i.e. the field name "x.y.z" says to check obj["x"]["y"]["z"].
func fieldMatches(field string, valueMatchers []string, obj map[string]interface{}) bool {
	val, ok := lookupString(field, obj)
	if !ok {
		return false
	}
	for _, match := range valueMatchers {
		if val == match {
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
