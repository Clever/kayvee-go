package middleware

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	kv "gopkg.in/Clever/kayvee-go.v3"
	"gopkg.in/Clever/kayvee-go.v3/logger"
)

type bufferWriter struct {
	bytes.Buffer
	status int
}

func (b *bufferWriter) WriteHeader(status int) {
	b.status = status
}

func (b *bufferWriter) Header() http.Header {
	return nil
}

func TestMiddleware(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		handler        func(http.ResponseWriter, *http.Request)
		expectedSize   int
		expectedStatus int
		expectedLog    map[string]interface{}
	}{
		{
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write(make([]byte, 10, 10))
				w.Write(make([]byte, 5, 5))
			},
			// Only the logs that vary based on the handler, the rest are tested in the test runner
			expectedLog: map[string]interface{}{
				"level": "info",
				// Floats because json decoding treats all numbers as floats
				"response-size": 15.0,
				"status-code":   200.0,
			},
		},
	}
	for _, test := range tests {
		lggr := logger.New("my-source")

		out := &bytes.Buffer{}
		lggr.SetConfig("my-source", logger.Info, kv.Format, out)

		handler := New(http.HandlerFunc(test.handler), lggr)
		rw := &bufferWriter{}
		handler.ServeHTTP(rw, &http.Request{
			Method: "GET",
			URL: &url.URL{
				Host:     "trollhost.com",
				Path:     "path",
				RawQuery: "key=val&key2=val2",
			},
			Header: http.Header{"X-Forwarded-For": {"192.168.0.1"}},
		})

		var result map[string]interface{}
		assert.Nil(json.NewDecoder(out).Decode(&result))

		log.Printf("%#v", result)

		// response-time changes each run, so just check that it's more than zero
		if result["response-time"].(float64) < 1 {
			t.Fatalf("invalid response-time %d", result["response-time"])
		}
		delete(result, "response-time")

		test.expectedLog["ip"] = "192.168.0.1"
		test.expectedLog["path"] = "path"
		test.expectedLog["method"] = "GET"
		test.expectedLog["title"] = "request-finished"
		test.expectedLog["via"] = "kayvee-middleware"
		test.expectedLog["params"] = "key=val&key2=val2"
		test.expectedLog["source"] = "my-source"
		test.expectedLog["deploy_env"] = "testing"

		assert.Equal(test.expectedLog, result)
	}
}
