Contributing to mysql-wear
-

## Developing Mysql-Wear

### Running the Tests

- Make sure mysql server is running
- Adjust mysql connection settings in conn_test.go (this is temporary. Later we will use a .env file or env vars)
- Use the Makefile to run the tests: `make test_all`

## Testing Notes

- You can simply run `go test -v inside this directory to run all the tests.
- See conn_test.go for the main test function and details
- Due to the way we configure the tmp/test db, running an individual test should be accomplished
  via `go test -v -run=TestName`
- Given the heavy use of structs, please have each test define its
  own structs inside the test function. Otherwise maintenance will become
  a real nightmare.

## Understanding Reflection

In order to improve this library a fairly in depth knowledge of golang's reflect
package is required. I would start with the following resources below, then try adding
a few tests around the parsing and Select code.

Resources (in order of importance)

- [Laws of Reflection by Rob Pike](https://blog.golang.org/laws-of-reflection)
  - Links to other posts
- Go book chapter ?
