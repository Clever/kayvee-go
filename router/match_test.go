package router

import (
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type SortableOutputs []map[string]interface{}

func (r SortableOutputs) Len() int {
	return len(r)
}
func (r SortableOutputs) Less(i, j int) bool {
	return r[i]["rule"].(string) < r[j]["rule"].(string)
}
func (r SortableOutputs) Swap(i, j int) {
	tmp := r[j]
	r[j] = r[i]
	r[i] = tmp
}

func TestMatchesSimple(t *testing.T) {
	r := Rule{
		Matchers: RuleMatchers{
			"title": []string{"hello", "hi"},
			"foo":   []string{"bar"},
		},
		Output: RuleOutput{},
	}
	msg0 := map[string]interface{}{
		"title": "hello",
		"foo":   "bar",
	}
	msg1 := map[string]interface{}{
		"title": "hi",
		"foo":   "bar",
	}
	msg2 := map[string]interface{}{
		"title": "hi",
		"foo":   "fighters",
	}
	msg3 := map[string]interface{}{
		"title": "howdy",
		"foo":   "bar",
	}
	assert.True(t, r.Matches(msg0))
	assert.True(t, r.Matches(msg1))
	assert.False(t, r.Matches(msg2))
	assert.False(t, r.Matches(msg3))
}

func TestMatchesNested(t *testing.T) {
	r := Rule{
		Matchers: RuleMatchers{
			"foo.bar": []string{"hello", "hi"},
		},
		Output: RuleOutput{},
	}
	msg0 := map[string]interface{}{
		"title": "greeting",
		"foo": map[string]interface{}{
			"bar": "hello",
		},
	}
	msg1 := map[string]interface{}{
		"title": "greeting",
		"foo": map[string]interface{}{
			"bar": "hi",
		},
	}
	msg2 := map[string]interface{}{
		"title": "greeting",
		"foo": map[string]interface{}{
			"bar": "howdy",
		},
	}
	assert.True(t, r.Matches(msg0))
	assert.True(t, r.Matches(msg1))
	assert.False(t, r.Matches(msg2))
}

func TestSubstitution(t *testing.T) {
	os.Setenv("KAYVEE_TEST_VAR", "howdy")
	r := Rule{
		Name:     "myrule",
		Matchers: RuleMatchers{},
		Output: RuleOutput{
			"channel":    "#${KAYVEE_TEST_VAR}-%{foo}-",
			"dimensions": []string{"-%{foo}-", "-%{bar.baz}-", "-${KAYVEE_TEST_VAR}-"},
		},
	}
	msg := map[string]interface{}{
		"title": "greeting",
		"foo":   "partner",
		"bar": map[string]interface{}{
			"baz": "nest egg",
		},
	}
	expected := map[string]interface{}{
		"rule":       "myrule",
		"channel":    "#howdy-partner-",
		"dimensions": []string{"-partner-", "-nest egg-", "-howdy-"},
	}
	actual := r.OutputFor(msg)
	assert.Equal(t, expected, actual)
}

func TestRoute(t *testing.T) {
	os.Setenv("KAYVEE_TEST_VAR", "howdy")
	router := RuleRouter{rules: []Rule{
		Rule{
			Name: "rule-one",
			Matchers: RuleMatchers{
				"title": []string{"hello", "hi"},
				"foo":   []string{"bar", "baz"},
			},
			Output: RuleOutput{
				"channel":    "#${KAYVEE_TEST_VAR}-%{foo}-",
				"dimensions": []string{"-%{foo}-"},
			},
		},
		Rule{
			Name: "rule-two",
			Matchers: RuleMatchers{
				"bing.bong": []string{"buzz"},
			},
			Output: RuleOutput{
				"series": "x",
			},
		},
	}}

	msg0 := map[string]interface{}{
		"title": "hi",
		"foo":   "bar",
	}
	expected0 := []map[string]interface{}{
		map[string]interface{}{
			"rule":       "rule-one",
			"channel":    "#howdy-bar-",
			"dimensions": []string{"-bar-"},
		},
	}
	actual0 := router.Route(msg0)["routes"].([]map[string]interface{})
	assert.Equal(t, expected0, actual0)

	msg1 := map[string]interface{}{
		"title": "hi",
		"bing": map[string]interface{}{
			"bong": "buzz",
		},
	}
	expected1 := []map[string]interface{}{
		map[string]interface{}{
			"rule":   "rule-two",
			"series": "x",
		},
	}
	actual1 := router.Route(msg1)["routes"].([]map[string]interface{})
	assert.Equal(t, expected1, actual1)

	msg2 := map[string]interface{}{
		"title": "hello",
		"foo":   "baz",
		"bing": map[string]interface{}{
			"bong": "buzz",
		},
	}
	expected2 := SortableOutputs([]map[string]interface{}{
		map[string]interface{}{
			"rule":       "rule-one",
			"channel":    "#howdy-baz-",
			"dimensions": []string{"-baz-"},
		},
		map[string]interface{}{
			"rule":   "rule-two",
			"series": "x",
		},
	})
	actual2 := SortableOutputs(router.Route(msg2)["routes"].([]map[string]interface{}))
	sort.Sort(expected2)
	sort.Sort(actual2)
	assert.Equal(t, expected2, actual2)
}