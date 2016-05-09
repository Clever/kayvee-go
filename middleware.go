package kayvee

import (
	"log"
	"net/http"
	"time"
)

var defaultHandler = func(req *http.Request) map[string]interface{} {
	return map[string]interface{}{
		"method": req.Method,
		"path":   req.URL.Path,
		"params": req.URL.RawQuery,
		"ip":     getIP(req),
	}
}

// A LogHandler is an http.Handler that logs customizable data about every request.
type LogHandler struct {
	source   string
	handlers []func(req *http.Request) map[string]interface{}
	h        http.Handler
}

func (l *LogHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	start := time.Now()

	lrw := &loggedResponseWriter{
		status:         -1,
		ResponseWriter: w,
		length:         0,
	}
	l.h.ServeHTTP(lrw, req)
	duration := time.Since(start)

	data := l.applyHandlers(req, map[string]interface{}{
		"response-time": duration,
		"response-size": lrw.length,
		"status-code":   lrw.status,
		"via":           "kayvee-middleware",
	})

	log.Println(FormatLog(l.source, logLevelFromStatus(lrw.status), "request-finished", data))
}

func (l *LogHandler) applyHandlers(req *http.Request, finalizer map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	writeData := func(data map[string]interface{}) {
		for key, val := range data {
			result[key] = val
		}
	}

	for _, handler := range l.handlers {
		writeData(handler(req))
	}
	// Write reserved fields last to make sure nothing overwrites them
	writeData(defaultHandler(req))
	writeData(finalizer)

	return result
}

// NewLogHandler takes in an http Handler to wrap with logging, the source of that logging, and any
// amount of optional handlers to customize the data that's logged.
func NewLogHandler(h http.Handler, source string, handlers ...func(*http.Request) map[string]interface{}) *LogHandler {
	return &LogHandler{
		source:   source,
		handlers: handlers,
		h:        h,
	}
}

// HeaderHandler takes in any amount of headers and returns a handler that adds those headers.
func HeaderHandler(headers ...string) func(*http.Request) map[string]interface{} {
	return func(req *http.Request) map[string]interface{} {
		result := map[string]interface{}{}
		for _, header := range headers {
			if val := req.Header.Get(header); val != "" {
				result[header] = val
			}
		}
		return result
	}
}

type loggedResponseWriter struct {
	status int
	http.ResponseWriter
	length int
}

func (w *loggedResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggedResponseWriter) Write(b []byte) (int, error) {
	if w.status == -1 {
		w.status = 200
	}
	n, err := w.ResponseWriter.Write(b)
	w.length += n
	return n, err
}

func getIP(req *http.Request) string {
	forwarded := req.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}
	return req.RemoteAddr
}

func logLevelFromStatus(status int) LogLevel {
	if status >= 499 {
		return Error
	}
	return Info
}
