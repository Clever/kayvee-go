package kayvee

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

type Tests struct {
	Version        string     `json:"version"`
	FormatTests    []TestSpec `json:"format"`
	FormatLogTests []TestSpec `json:"formatLog"`
}

type TestSpec struct {
	Title  interface{}            `json:"title"`
	Input  map[string]interface{} `json:"input"`
	Output interface{}            `json:"output"`
}

type keyVal map[string]interface{}

func Test_KayveeSpecs(t *testing.T) {
	file, err := ioutil.ReadFile("tests.json")
	assert.NoError(t, err, "failed to open test specs (tests.json)")
	var tests Tests
	json.Unmarshal(file, &tests)
	t.Logf("spec (tests.json) version: %s\n", string(tests.Version))

	for _, spec := range tests.FormatTests {
		expected := spec.Output
		actual := Format(spec.Input["data"].(map[string]interface{}))
		assert.Equal(t, expected, actual)
	}

	for _, spec := range tests.FormatLogTests {
		expected := spec.Output

		// Ensure correct type is passed to FormatLog, even if value undefined in tests.json
		source, _ := spec.Input["source"].(string)
		level, _ := spec.Input["level"].(string)
		title, _ := spec.Input["title"].(string)
		data, _ := spec.Input["data"].(map[string]interface{})
		actual := FormatLog(source, level, title, data)
		assert.Equal(t, expected, actual)
	}

}
