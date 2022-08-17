# gocmd
In the Go world, we can install several versions of Go without **\*\*env** by using [wrapper program](https://cs.opensource.google/go/dl).
```shell
$ go install golang.org/dl/go1.19@latest
$ go1.19 download
```
With IDEs like Visual Studio Code or GoLand, we can choose version of "go" command and use it without typing "go1.19".
Because their embedded terminal emulators set `PATH` environment variables automatically.

But in other environments, we have to type "go1.19" to use go command of the version "go1.19".
When creating devtools using go command, this differed behavior is not preferable.

So, in order to use expected version of go, following utilities are needed.

----

## Validate Go version
All released version is read from [here](https://go.dev/dl/?mode=json&include=all).
```go
err := ValidateVersion("go1.19")
// err == nil

err = ValidateVersion("unknown")
// err == ErrInvalidVersion
```

## Check the version of "go" command whether it matches to the version written in "go.mod"
Get the version of "go" command in your environment and the version written in go.mod, and compare them.
```
module m

go 1.18
```
```go
err := ValidModuleGoVersion("go1.18.5")
// err == nil

err = ValidModuleGoVersion("go1.17")
// err == ErrUnexpectedVersion
```

## Get the path of "go" executable that has the given version
```shell
$ go env GOVERSION
go1.19
$ which go1.18.6
/Users/me/go/bin/go1.18.5
$ which go1.17.5
go1.17.5 not found
```
```go
path, err := Lookup("go1.19")
// path == "go"
// err == nil

path, err = Lookup("go1.18.5")
// path == "/Users/me/go/bin/go1.18.5"
// err == nil

path, err = Lookup("go1.17.5")
// path == ""
// err == ErrNotFound
```
