package router

import (
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
	lookup := func(field string) (string, bool) {
		return lookupString(field, msg)
	}
	subbed := substituteFields(r.Output, lookup)
	subbed["rule"] = r.Name
	return subbed
}

// lookupString does an extended lookup on `obj`, interpreting dots in field as
// corresponding to subobjects. It returns the value and true if the lookup
// succeeded or `"", false` if the key is missing or corresponds to a
// non-string value.
func lookupString(field string, obj map[string]interface{}) (string, bool) {
	if strings.Index(field, ".") == -1 {
		val, ok := obj[field].(string)
		return val, ok
	}
	return lookupStringPath(strings.Split(field, "."), obj)
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
