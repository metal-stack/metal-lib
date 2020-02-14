# zapup

Zapup is an initializer around [ZAP](https://github.com/uber-go/zap/). 

It was intentionally created to provide a zap logger instance that considers:

- log output format
- log level
- provides file and line number of caller

## Quickstart

Enable go modules and go proxy to fetch the sources from `git.f-i-ts.de`:

```bash
export GO111MODULE=on
export GOPROXY=https://gomods.fi-ts.io
```

Create a `zapuptest` app:

```go
mkdir zapuptest
cd zapuptest
go mod init
```

Use zapup, e.g in `main.go``:

```go
package main

import "github.com/metal-stack/metal-lib/zapup"

var log = zapup.MustRootLogger()

func main() {
	log.Info("Hello, metal!")
}

```

Compile and run:

```go
go run main.go
```

Expected output:

```bash
[...]
{"level":"info","ts":1540279941.106842,"caller":"zapuptest/main.go:8","msg":"Hello, metal!","app":"","customer":"","stage":""}

```

## Configuration

Zapup can be configured by configuring environment variables. All of them have defaults:

```bash
#  Sets the log level. Valid values: "info", "debug", "warn", "error", "dpanic", "panic", "fatal". Defaults to "info".
export ZAP_LEVEL=debug
# Set the log encoding. Valid values: "console", "json". Defaults to "json".
export ZAP_ENCODING=console 
# Sets the field "customer" to annotate the log for usage e.g. within elasticsearch. Defaults to empty string.
export ZAP_CUSTOMER=""
# Sets the field "app" to annotate the log for usage e.g. within elasticsearch. Defaults to empty string.
export ZAP_APP=""
# Sets the field "stage" to annotate the log for usage e.g. within elasticsearch. Defaults to empty string.
export ZAP_STAGE="" 
# Sets the output path to write logs to. Valid values: "stdout", "stderr", "<file path>". Defaults to "stdout"
export ZAP_OUT="stdout"
```

## Usage

See [Zap](https://godoc.org/go.uber.org/zap).

But Obacht! Instead of:

```go
sugar := zap.NewExample().Sugar()
```

Run:

```go
sugar := zapup.MustRootLogger().Sugar()
```