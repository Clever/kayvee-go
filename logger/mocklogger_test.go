package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	router "gopkg.in/Clever/kayvee-go.v5/router"
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
	testRouter := router.NewFromRoutes(routes)

	mockLogger := NewMockCountLogger("testing")
	mockLogger.logger.logRouter = testRouter

	data0 := map[string]interface{}{
		"wrong": "stuff",
	}
	expected0 := map[string]int{}
	mockLogger.InfoD("log0", data0)
	actual0 := mockLogger.RuleCounts()
	assert.Equal(t, expected0, actual0)

	data1 := map[string]interface{}{
		"foo": "bar",
	}
	expected1 := map[string]int{
		"rule-one": 1,
	}
	mockLogger.InfoD("log1", data1)
	actual1 := mockLogger.RuleCounts()
	assert.Equal(t, expected1, actual1)

	data2 := map[string]interface{}{
		"foo": "bar",
		"abc": "def",
	}
	expected2 := map[string]int{
		"rule-one": 2,
		"rule-two": 1,
	}
	mockLogger.InfoD("log2", data2)
	actual2 := mockLogger.RuleCounts()
	assert.Equal(t, expected2, actual2)
}
