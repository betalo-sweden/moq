**Note**: This is a modified version of [matryer/moq](https://github.com/matryer/moq).
The following major modifications were applied:

- Declare package import paths using the non-vendored notation; allowing the Go
  compiler to compile the mock source files without any further manual
  post-editing; including support for build tags. ([ca23463](https://github.com/betalo-sweden/moq/commit/ca234637392db0e9d4d9fde5235df22b8cbfcafb), [1faeabd](https://github.com/betalo-sweden/moq/commit/1faeabd073f8d381acdfde1cbbebb96f06188cf5), [44c3e2a](https://github.com/betalo-sweden/moq/commit/44c3e2a5c504dc913411d111958d3a49830e8c4b), [782c95e](https://github.com/betalo-sweden/moq/commit/782c95e037a1981e46810e0988499a7220fe32c4))
- Apply `goimports` (instead of `gofmt`) on generated source; allowing
  configured strict linters to accept the mock source files. ([ed4df3d](https://github.com/betalo-sweden/moq/commit/ed4df3d6768318b5d2ee1d91b9fc4f0807724875))
- Support same package name for mock and types used in mock([62e06d1](https://github.com/betalo-sweden/moq/commit/62e06d143014a75b377ff01f33031319f289637b), [c20b2d5](https://github.com/betalo-sweden/moq/commit/c20b2d54c6140ca36ffddf9edafb3c3955f16f71))
- Support empty `GOPATH` ([a85236d](https://github.com/betalo-sweden/moq/commit/a85236d62cfe82405ef02e83ada392d54727a9ba))
- Assert that mock implementation always fully satisfies the interface.
  ([a870503](https://github.com/betalo-sweden/moq/commit/a87050393d8a6432efb45017a8ee1eef59d3248d))
- Generate non-executable go source files; addressing a potential security risk.
  ([8385b56](https://github.com/betalo-sweden/moq/commit/8385b56848247e389b8641a5d5ed324aff93430d))
- Remove tool name from `panic` output; reducing a reader's confusion when
  `panic`s occur. ([a781a2e](https://github.com/betalo-sweden/moq/commit/a781a2eb03616356cb1fcaf3d6962dc4599959ee))


# moq

![moq logo](moq-logo-small.png) [![Build Status](https://travis-ci.org/matryer/moq.svg?branch=master)](https://travis-ci.org/matryer/moq) [![Go Report Card](https://goreportcard.com/badge/github.com/matryer/moq)](https://goreportcard.com/report/github.com/matryer/moq)

Interface mocking tool for go generate.

By [Mat Ryer](https://twitter.com/matryer) and [David Hernandez](https://github.com/dahernan), with ideas lovingly stolen from [Ernesto Jimenez](https://github.com/ernesto-jimenez).

### What is Moq?

Moq is a tool that generates a struct from any interface. The struct can be used in test code as a mock of the interface.

![Preview](preview.png)

above: Moq generates the code on the right.

You can read more in the [Meet Moq blog post](http://bit.ly/meetmoq).

### Installing

To start using Moq, just run go get:
```
$ go get github.com/matryer/moq
```

### Usage

```
moq [flags] destination interface [interface2 [interface3 [...]]]
  -out string
    	output file (default stdout)
  -pkg string
    	package name (default will infer)
```

In a command line:

```
$ moq -out mocks_test.go . MyInterface
```

In code (for go generate):

```go
package my

//go:generate moq -out myinterface_moq_test.go . MyInterface

type MyInterface interface {
	Method1() error
	Method2(i int)
}
```

Then run `go generate` for your package.

### How to use it

Mocking interfaces is a nice way to write unit tests where you can easily control the behaviour of the mocked object.

Moq creates a struct that has a function field for each method, which you can declare in your test code.

This this example, Moq generated the `EmailSenderMock` type:

```go
func TestCompleteSignup(t *testing.T) {

	var sentTo string

	mockedEmailSender = &EmailSenderMock{
		SendFunc: func(to, subject, body string) error {
			sentTo = to
			return nil
		},
	}

	CompleteSignUp("me@email.com", mockedEmailSender)

	callsToSend := len(mockedEmailSender.SendCalls())
	if callsToSend != 1 {
		t.Errorf("Send was called %d times", callsToSend)
	}
	if sentTo != "me@email.com" {
		t.Errorf("unexpected recipient: %s", sentTo)
	}

}

func CompleteSignUp(to string, sender EmailSender) {
	// TODO: this
}
```

The mocked structure implements the interface, where each method calls the associated function field.

## Tips

* Keep mocked logic inside the test that is using it
* Only mock the fields you need
* It will panic if a nil function gets called
* Name arguments in the interface for a better experience
* Use closured variables inside your test function to capture details about the calls to the methods
* Use `.MethodCalls()` to track the calls
* Use `go:generate` to invoke the `moq` command

## License

The Moq project (and all code) is licensed under the [MIT License](LICENSE).

The Moq logo was created by [Chris Ryer](http://chrisryer.co.uk) and is licensed under the [Creative Commons Attribution 3.0 License](https://creativecommons.org/licenses/by/3.0/).
