package router

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type SortableRules []Rule

func (r SortableRules) Len() int {
	return len(r)
}
func (r SortableRules) Less(i, j int) bool {
	return r[i].Name < r[j].Name
}
func (r SortableRules) Swap(i, j int) {
	tmp := r[j]
	r[j] = r[i]
	r[i] = tmp
}

func TestParsesWellFormatedConfig(t *testing.T) {
	conf := []byte(`
routes:
  rule-one:
    matchers:
      title: ["authorize-app"]
    output:
      type: "notification"
      channel: "#team"
      icon: ":rocket:"
      message: "authorized %{foo.bar} in ${SCHOOL}"
      user: "@fishman"
  rule-two:
    matchers:
      foo.bar: ["multiple", "matches"]
      baz: ["whatever"]
    output:
      type: "alert"
      series: "other-series"
      dimensions: ["baz"]
      stat_type: "gauge"
`)
	expected := SortableRules{
		Rule{
			Name:     "rule-one",
			Matchers: RuleMatchers{"title": []string{"authorize-app"}},
			Output: RuleOutput{
				"type":    "notification",
				"channel": "#team",
				"icon":    ":rocket:",
				"message": `authorized %{foo.bar} in Hogwarts`,
				"user":    "@fishman",
			},
		},
		Rule{
			Name: "rule-two",
			Matchers: RuleMatchers{
				"foo.bar": []string{"multiple", "matches"},
				"baz":     []string{"whatever"},
			},
			Output: RuleOutput{
				"type":       "alert",
				"series":     "other-series",
				"dimensions": []interface{}{"baz"},
				"stat_type":  "gauge",
			},
		},
	}

	err := os.Setenv("SCHOOL", "Hogwarts")
	assert.Nil(t, err)

	router, err := newFromConfigBytes(conf)
	assert.Nil(t, err)

	actual := SortableRules(router.rules)
	sort.Sort(expected)
	sort.Sort(actual)
	assert.Equal(t, expected, actual)
}

func TestOnlyStringMatcherValues(t *testing.T) {
	confTmpl := `
routes:
  non-string-values:
    matchers:
      no-numbers: [%s]
    output:
      type: "analytics"
      series: "fun"
`

	// Make sure the template works
	conf := []byte(fmt.Sprintf(confTmpl, "\"valid\""))
	_, err := newFromConfigBytes(conf)
	assert.Nil(t, err)

	for _, invalidVal := range []string{"5", "true", "[]", "{}"} {
		conf := []byte(fmt.Sprintf(confTmpl, invalidVal))
		_, err := newFromConfigBytes(conf)
		assert.Error(t, err)
	}
}

func TestNoSpecialsInMatcher(t *testing.T) {
	confFieldTmpl := `
routes:
  complicated-fields:
    matchers:
      "%s": ["hallo?"]
    output:
      type: "analytics"
      series: "fun"
`
	confValTmpl := `
routes:
  complicated-values:
    matchers:
      title: ["%s"]
    output:
      type: "analytics"
      series: "fun"
`

	// Make sure templates work
	for _, tmpl := range []string{confFieldTmpl, confValTmpl} {
		conf := []byte(fmt.Sprintf(tmpl, "valid"))
		_, err := newFromConfigBytes(conf)
		assert.Nil(t, err)
	}

	invalids := []string{"${wut}", "%{wut}", "$huh", "}ok?", "nope{", `100% fail`}
	for _, invalid := range invalids {
		for _, tmpl := range []string{confFieldTmpl, confValTmpl} {
			conf := []byte(fmt.Sprintf(tmpl, invalid))
			_, err := newFromConfigBytes(conf)
			assert.Error(t, err)
		}
	}
}

func TestNoDupMatchers(t *testing.T) {
	confTmpl := `
routes:
  sloppy:
    matchers:
      title: [%s]
    output:
      type: "analytics"
      series: "fun"
`

	validConf := []byte(fmt.Sprintf(confTmpl, `"non-repeated", "name"`))
	_, err := newFromConfigBytes(validConf)
	assert.Nil(t, err)

	invalidConf := []byte(fmt.Sprintf(confTmpl, `"repeated", "repeated", "name"`))
	_, err = newFromConfigBytes(invalidConf)
	assert.Error(t, err)
}

func TestOutputRequiresCorrectTypes(t *testing.T) {
	confTmpl := `
routes:
  wrong:
    matchers:
      title: ["test"]
    output:
      type: "alert"
      series: %s
      dimensions: %s
      stat_type: "gauge"
`

	validConf := []byte(fmt.Sprintf(confTmpl, `"my-series"`, `["dim1", "dim2"]`))
	_, err := newFromConfigBytes(validConf)
	assert.Nil(t, err)

	invalidConf0 := []byte(fmt.Sprintf(confTmpl, `["my-series"]`, `["dim1", "dim2"]`))
	_, err = newFromConfigBytes(invalidConf0)
	assert.Error(t, err)

	invalidConf1 := []byte(fmt.Sprintf(confTmpl, `"my-series"`, `"dim1"`))
	_, err = newFromConfigBytes(invalidConf1)
	assert.Error(t, err)
}

func TestOutputRequiresAllKeys(t *testing.T) {
	confTmpl := `
routes:
  wrong:
    matchers:
      title: ["test"]
    output:
      type: "alert"%s
      dimensions: ["dim1", "dim2"]
      stat_type: "gauge"
`

	validConf := []byte(fmt.Sprintf(confTmpl, `
      series: "whatever"`))
	_, err := newFromConfigBytes(validConf)
	assert.Nil(t, err)

	invalidConf := []byte(fmt.Sprintf(confTmpl, ``))
	_, err = newFromConfigBytes(invalidConf)
	assert.Error(t, err)
}

func TestOutputNoExtraKeysAllowed(t *testing.T) {
	confTmpl := `
routes:
  wrong:
    matchers:
      title: ["test"]
    output:
      type: "alert"%s
      dimensions: ["dim1", "dim2"]
      stat_type: "gauge"
`

	validConf := []byte(fmt.Sprintf(confTmpl, `
      series: "whatever"`))
	_, err := newFromConfigBytes(validConf)
	assert.Nil(t, err)

	invalidConf := []byte(fmt.Sprintf(confTmpl, `
      series: "whatever"
      something-else: "hi there"`))
	_, err = newFromConfigBytes(invalidConf)
	assert.Error(t, err)
}
