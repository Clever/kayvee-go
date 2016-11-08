# kayvee
--
    import "gopkg.in/Clever/kayvee-go.v5"

Package kayvee provides methods to output human and machine parseable strings,
with a "json" format.

## [Logger API Documentation](./logger)

* [gopkg.in/Clever/kayvee-go.v5/logger](https://godoc.org/gopkg.in/Clever/kayvee-go.v5/logger)
* [gopkg.in/Clever/kayvee-go.v5/middleware](https://godoc.org/gopkg.in/Clever/kayvee-go.v5/middleware)

## Example

```go
    package main

    import(
        "fmt"
        "time"

        "gopkg.in/Clever/kayvee-go.v5/logger"
    )

    var log = logger.New("myApp")

    func main() {
        // Simple debugging
        log.Debug("Service has started")

        // Make a query and log its length
        query_start := time.Now()
        log.GaugeFloat("QueryTime", time.Since(query_start).Seconds())

        // Output structured data
        log.InfoD("DataResults", logger.M{"key": "value"})

        // You can use the M alias for your key value pairs
        log.InfoD("DataResults", logger.M{"shorter": "line"})
    }
```


## Testing

Run `make test` to execute the tests

## Change log

- v5.0 - Middleware logger now creates a new logger on each request.
  - Breaking change to `middleware.New` constructor.
- v4.0
  - Added methods to read and write the `Logger` object from a a `context.Context` object.
  - Middleware now injects the logger into the request context.
  - Updated to require Go 1.7.
- v4.0 - Removed sentry-go dependency
- v2.4 - Add kayvee-go/validator for asserting that raw log lines are in a valid kayvee format.
- v2.3 - Expose logger.M.
- v2.2 - Remove godeps.
- v2.1 - Add kayvee-go/logger with log level, counters, and gauge support
- v0.1 - Initial release.

## Backward Compatibility

The kayvee 1.x interface still exist but is considered deprecated. You can find documentation on using it in the [compatibility guide](./compatibility.md)

## Publishing

To release a new version run `make bump-major`, `make bump-minor`, or `make
bump-patch` as appropriate on master (after merging your PR). Then, run `git
push --tags`.
