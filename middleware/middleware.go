/*
Package middleware provides a customizable Kayvee logging middleware for HTTP servers.

	logHandler := New(myHandler, myLogger, func(req *http.Request) map[string]interface{} {
		// Add Gorilla mux vars to the log, just because
		return mux.Vars(req)
	})
*/
package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Clever/kayvee-go/v7/logger"
)

var defaultHandler = func(req *http.Request) map[string]interface{} {
	data := map[string]interface{}{
		"method": req.Method,
		"path":   req.URL.Path,
		"params": req.URL.RawQuery,
		"ip":     getIP(req),
	}

	// TODO: wag should inject metadata into the req context
	// Then we wouldn't need to expose logger globals via GetContext
	if op, ok := logger.FromContext(req.Context()).GetContext("op"); ok {
		data["op"] = op
	}
	return data
}

type logHandler struct {
	handlers []func(req *http.Request) map[string]interface{}
	h        http.Handler
	source   string
	ip       string
}

func (l *logHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	start := time.Now()

	// create and inject a logger into req.Context
	lggr := logger.NewConcreteLogger(l.source)
	// This allows us to more easily correlate application log with alb
	// access logs.
	lggr.AddContext("task_ip", l.ip)
	req = req.WithContext(logger.NewContext(req.Context(), lggr))

	lrw := &loggedResponseWriter{
		status:         200,
		ResponseWriter: w,
		length:         0,
	}
	l.h.ServeHTTP(lrw, req)
	duration := time.Since(start)

	data := l.applyHandlers(req, map[string]interface{}{
		"response-time":    duration,
		"response-time-ms": duration.Nanoseconds() / int64(time.Millisecond),
		"count":            1, // this makes aggregating single logs with rollup logs easier
		"response-size":    lrw.length,
		"status-code":      lrw.status,
		"via":              "kayvee-middleware",
	})

	// check if the user has opted in to rolling up middleware logs
	if globalRollupRouter != nil && globalRollupRouter.ShouldRollup(data) {
		globalRollupRouter.Process(data)
		return
	}

	switch logLevelFromStatus(lrw.status) {
	case logger.Error:
		lggr.ErrorD("request-finished", data)
	case logger.Warning:
		lggr.WarnD("request-finished", data)
	default:
		lggr.InfoD("request-finished", data)
	}
}

func (l *logHandler) applyHandlers(req *http.Request, finalizer map[string]interface{}) map[string]interface{} {
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

// New takes in an http Handler to wrap with logging, the logger source name to use, and any amount of
// optional handlers to customize the data that's logged.
// On every request, the middleware will create a logger and place it in req.Context().
func New(h http.Handler, source string, handlers ...func(*http.Request) map[string]any) http.Handler {
	ip, err := getTaskIP()
	if err != nil {
		log.Println("ERROR: couldn't get task IP:", err)
	}
	return &logHandler{
		handlers: handlers,
		h:        h,
		source:   source,
		ip:       ip,
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

func logLevelFromStatus(status int) logger.LogLevel {
	if status >= 499 {
		return logger.Error
	}
	if status >= 400 {
		return logger.Warning
	}
	return logger.Info
}

// TaskMetadata represents the ECS task metadata structure
type taskMetadata struct {
	Cluster    string      `json:"Cluster"`
	TaskARN    string      `json:"TaskARN"`
	Family     string      `json:"Family"`
	Revision   string      `json:"Revision"`
	Containers []container `json:"Containers"`
}

type container struct {
	Name     string    `json:"Name"`
	Image    string    `json:"Image"`
	Networks []network `json:"Networks"`
}

type network struct {
	NetworkMode   string   `json:"NetworkMode"`
	IPv4Addresses []string `json:"IPv4Addresses"`
}

func getTaskIP() (string, error) {
	metadataURI := os.Getenv("ECS_CONTAINER_METADATA_URI_V4")
	if metadataURI == "" {
		return "", fmt.Errorf("ECS_CONTAINER_METADATA_URI_V4 not set")
	}

	resp, err := http.Get(metadataURI + "/task")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var taskMetadata taskMetadata
	if err := json.Unmarshal(body, &taskMetadata); err != nil {
		return "", err
	}

	cn := fmt.Sprintf("%s--%s", os.Getenv("_DEPLOY_ENV"), os.Getenv("_APP_NAME"))

	// Find the specific container
	for _, container := range taskMetadata.Containers {
		if container.Name == cn {
			if len(container.Networks) > 0 && len(container.Networks[0].IPv4Addresses) > 0 {
				return container.Networks[0].IPv4Addresses[0], nil
			}
		}
	}

	return "", fmt.Errorf("container %s not found or has no IP", cn)
}
