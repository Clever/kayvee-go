# kayvee
--
    import "gopkg.in/clever/kayvee-go.v2"

Package kayvee provides methods to output human and machine parseable strings,
with a "json" format.

## [Logger Documentation](./logger)

## Example

Here's an example program that outputs a kayvee formatted string:

    package main

    import(
      "fmt"
      "gopkg.in/Clever/kayvee-go.v2"
    )

    func main() {
      fmt.Println(kayvee.Format(map[string]interface{}{"hello": "world"}))
    }

## Testing


Run `make test` to execute the tests

## Change log

v2.1 - Add kayvee-go/logger with log level, counters, and gauge support
v0.1 - Initial release.

## Usage

#### func  Format

```go
func Format(data map[string]interface{}) string
```
Format converts a map to a string of space-delimited key=val pairs

#### func  FormatLog

```go
func FormatLog(source string, level LogLevel, title string, data map[string]interface{}) string
```
FormatLog is similar to Format, but takes additional reserved params to promote
logging best-practices

#### type LogLevel

```go
type LogLevel string
```

LogLevel denotes the level of a logging

```go
const (
	Unknown  LogLevel = "unknown"
	Critical          = "critical"
	Error             = "error"
	Warning           = "warning"
	Info              = "info"
	Trace             = "trace"
)
```
Constants used to define different LogLevels supported

#### type Logger

```go
type Logger interface {
	Info(title string, data map[string]interface{})
	Warning(title string, data map[string]interface{})
	Error(title string, data map[string]interface{}, err error)
}
```

Logger is an interface satisfied by all loggers that use kayvee to Log results

#### type SentryLogger

```go
type SentryLogger struct {
}
```

SentryLogger provides an wrapper methods to do logging using kayvee and
optionally sending errors to kayvee.

#### func  NewSentryLogger

```go
func NewSentryLogger(source string, logger *log.Logger, sentryClient *raven.Client) *SentryLogger
```
NewSentryLogger returns a new *kayvee.Logger. source is the value assigned for
all logs generated by the logger log.Logger is the underlying logger used. If
nil, uses log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lshortfile)
sentryClient is used to optionally route errors to sentry.

#### func (*SentryLogger) Error

```go
func (l *SentryLogger) Error(title string, data map[string]interface{}, err error)
```
Error writes a log with level kayvee.Error If the logger was initialized with a
sentryClient and error is not nil, captures the error for sentry and assigns the
event ID to the `sentry_event_id` key in the data.

#### func (*SentryLogger) Info

```go
func (l *SentryLogger) Info(title string, data map[string]interface{})
```
Info writes a log with level kayvee.Info

#### func (*SentryLogger) Warning

```go
func (l *SentryLogger) Warning(title string, data map[string]interface{})
```
Warning writes a log with level kayvee.Warning
