# kayvee
--
Package kayvee provides methods to output human and machine parseable strings,
with a "key=val" format.

## Example

Here's an example program that outputs a kayvee formatted string:

    package main

    import(
      "fmt"
      "gopkg.in/Clever/kayvee.v0"
    )

    func main() {
      fmt.Println(kayvee.Format(map[string]interface{}{"hello": "world"}))
    }

Here's an example of using the logger in Kayvee

    import kv "gopkg.in/Clever/kayvee.v3"

    kv.WithFields(kv.F{"key", "value"}).Warning("yellow alert")
    # outputs


## Testing



## Testing


Run `make test` to execute the tests

## Change log

v0.0.1 - Initial release.
