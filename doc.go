/*

Package kayvee provides methods to output human and machine parseable strings, with a "key=val" format.

## Example

Here's an example program that outputs a kayvee formatted string:

  package main

  import(
    "fmt"
    "github.com/Clever/kayvee-go/v7"
  )

  func main() {
    fmt.Println(kayvee.Format(map[string]interface{}{"hello": "world"}))
  }

## Testing

Run `make test` to execute the tests

## Change log

v0.0.1 - Initial release.

*/
package kayvee
