# kayvee
--
    import "gopkg.in/clever/kayvee-go.v1"


## Usage

#### func  Format

```go
func Format(data map[string]interface{}) string
```
Format converts a map to a string of space-delimited key=val pairs

#### func  FormatLog

```go
func FormatLog(source string, level string, title string, data map[string]interface{}) string
```
FormatLog is similar to Format, but takes additional reserved params to promote
logging best-practices
