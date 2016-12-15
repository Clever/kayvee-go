package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	router "gopkg.in/Clever/kayvee-go.v6/router"
)

func TestMockLoggerImplementsKayveeLogger(t *testing.T) {
	assert.Implements(t, (*KayveeLogger)(nil), &MockRouteCountLogger{}, "*MockRouteCountLogger should implement KayveeLogger")
}

func TestRouteCountsWithMockLogger(t *testing.T) {
	routes := map[string](router.Rule){
		"rule-one": router.Rule{
			Matchers: router.RuleMatchers{
				"foo": []string{"bar", "baz"},
			},
			Output: router.RuleOutput{
				"out": "#-%{foo}-",
			},
		},
		"rule-two": router.Rule{
			Matchers: router.RuleMatchers{
				"abc": []string{"def"},
			},
			Output: router.RuleOutput{
				"more": "x",
			},
		},
	}
	testRouter, err := router.NewFromRoutes(routes)
	assert.NoError(t, err)

	mockLogger := NewMockCountLogger("testing")
	mockLogger.SetRouter(testRouter)

	t.Log("log0")
	data0 := M{
		"wrong": "stuff",
	}
	mockLogger.InfoD("log0", data0)

	t.Log("log0 -- verify rule counts")
	actualCounts0 := mockLogger.RuleCounts()
	expectedCounts0 := map[string]int{}
	assert.Equal(t, expectedCounts0, actualCounts0)

	t.Log("log0 -- verify rule matches")
	actualMatches0 := mockLogger.RuleMatches()
	expectedMatches0 := map[string][]M{}
	assert.Equal(t, expectedMatches0, actualMatches0)

	t.Log("log1")
	data1 := M{
		"foo": "bar",
	}
	mockLogger.InfoD("log1", data1)

	t.Log("log1 -- verify rule counts")
	actualCounts1 := mockLogger.RuleCounts()
	expectedCounts1 := map[string]int{"rule-one": 1}
	assert.Equal(t, expectedCounts1, actualCounts1)

	t.Log("log1 -- verify rule matches")
	expectedRoutedLog1 := M{
		"foo":        "bar",
		"source":     "testing",
		"title":      "log1",
		"level":      "info",
		"deploy_env": "testing",
		"_kvmeta": map[string]interface{}{
			"kv_language": "go",
			"kv_version":  "6.2.0",
			"team":        "UNSET",
			"routes": []map[string]interface{}{
				map[string]interface{}{
					"rule": "rule-one",
					"out":  "#-bar-",
				},
			},
		},
	}
	actualMatches1 := mockLogger.RuleMatches()
	expectedMatches1 := map[string][]M{
		"rule-one": []M{expectedRoutedLog1},
	}
	assert.Equal(t, expectedMatches1, actualMatches1)

	t.Log("log2")
	data2 := M{
		"foo": "bar",
		"abc": "def",
	}
	mockLogger.InfoD("log2", data2)

	t.Log("log2 -- verify rule counts")
	actualCounts2 := mockLogger.RuleCounts()
	expectedCounts2 := map[string]int{
		"rule-one": 2,
		"rule-two": 1,
	}
	assert.Equal(t, expectedCounts2, actualCounts2)

	t.Log("log2 -- verify rule matches")
	expectedRoutedLog2 := M{
		"foo":        "bar",
		"abc":        "def",
		"source":     "testing",
		"title":      "log2",
		"level":      "info",
		"deploy_env": "testing",
		"_kvmeta": map[string]interface{}{
			"kv_language": "go",
			"kv_version":  "6.2.0",
			"team":        "UNSET",
			"routes": []map[string]interface{}{
				map[string]interface{}{
					"rule": "rule-one",
					"out":  "#-bar-",
				},
				map[string]interface{}{
					"rule": "rule-two",
					"more": "x",
				},
			},
		},
	}
	expectedMatches2 := map[string][]M{
		"rule-one": []M{
			expectedRoutedLog1,
			expectedRoutedLog2,
		},
		"rule-two": []M{
			expectedRoutedLog2,
		},
	}

	actualMatches2 := mockLogger.RuleMatches()
	assert.Equal(t, expectedMatches2, actualMatches2)
}
