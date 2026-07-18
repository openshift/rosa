# consolesize-go

[![Sponsor Me!](https://img.shields.io/badge/%F0%9F%92%B8-Sponsor%20Me!-blue)](https://github.com/sponsors/nathan-fiscaletti)
[![GoDoc](https://godoc.org/github.com/nathan-fiscaletti/consolesize-go?status.svg)](https://godoc.org/github.com/nathan-fiscaletti/consolesize-go)

A tiny, dependency-free Go library for reading **terminal width and height** (columns × rows) on the machine your process is attached to.

- **No third-party modules** — only the standard library  
- **Linux, macOS, BSDs, and Windows** — one import, two files behind build tags  
- **~100 lines of Go** — uses ioctl on Unix, `GetConsoleScreenBufferInfo` on Windows  

## Used by

- Microsoft projects  ([Azure/azure-dev](https://github.com/Azure/azure-dev))
- AWS-related tooling  ([awslabs/diagram-as-code](https://github.com/awslabs/diagram-as-code), [aws-cloudformation/rain](https://github.com/aws-cloudformation/rain))
- **200+** public repositories 

## Why not `golang.org/x/term`?

`x/term` (and the wider `x/` family) is the right choice when you want a full terminal feature set — raw mode, bracketed paste, cursor APIs, and more. It pulls in **`golang.org/x/sys`** and a larger API surface.

**consolesize-go** is for when you only need **window size**: one function, no extra module graph, and nothing else to learn or audit. That is a deliberate trade-off, not a knock on `x/term`.

## Install

```sh
go get github.com/nathan-fiscaletti/consolesize-go
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/nathan-fiscaletti/consolesize-go"
)

func main() {
	cols, rows := consolesize.GetConsoleSize()
	fmt.Printf("columns: %d, rows: %d\n", cols, rows)
}
```

`GetConsoleSize` returns **(columns, rows)**.

## Binary size

Minimal mains that only read terminal size are built side-by-side under [`sizecmp/`](sizecmp/): one links **this module**, the other links **`golang.org/x/term`** (which pulls **`x/sys`**). Both use the same `go build` flags.

The script strips symbols (`-ldflags=-s -w`) and uses `-trimpath` so paths on your machine do not affect the build. It prints **byte sizes** and the **delta** for your `GOOS`/`GOARCH` and Go toolchain. Exact numbers change across releases; the typical pattern is a **smaller** binary when you avoid `x/term` + `x/sys` for size-only callers. On an Apple M5 this saves **~17kb**.

```sh
./sizecmp/compare.sh

go version: go version go1.26.1 darwin/arm64
GOOS=darwin GOARCH=arm64
flags: go build -ldflags=-s -w -trimpath

consolesize-go  1454754 bytes
golang.org/x/term  1471874 bytes
delta (x/term - consolesize-go)  +17120 byte
```

## Known projects using consolesize-go

**Based on**: [network/dependents](https://github.com/nathan-fiscaletti/consolesize-go/network/dependents) as of April 6th, 2026.

- [xo/usql](https://github.com/xo/usql)
- [TheZoraiz/ascii-image-converter](https://github.com/TheZoraiz/ascii-image-converter)
- [xyproto/algernon](https://github.com/xyproto/algernon)
- [neilotoole/sq](https://github.com/neilotoole/sq)
- [fwdcloudsec/granted](https://github.com/fwdcloudsec/granted)
- [awslabs/diagram-as-code](https://github.com/awslabs/diagram-as-code)
- [aws-cloudformation/rain](https://github.com/aws-cloudformation/rain)
- [xyproto/orbiton](https://github.com/xyproto/orbiton)
- [Azure/azure-dev](https://github.com/Azure/azure-dev)
- [pingcap/tiup](https://github.com/pingcap/tiup)
- [mondoohq/mql](https://github.com/mondoohq/mql)
- [common-fate/glide](https://github.com/common-fate/glide)
- [viamrobotics/rdk](https://github.com/viamrobotics/rdk)
- [xyproto/gendesk](https://github.com/xyproto/gendesk)
- [Isan-Rivkin/surf](https://github.com/Isan-Rivkin/surf)
- [arimatakao/mdx](https://github.com/arimatakao/mdx)
- [openshift/rosa](https://github.com/openshift/rosa)

## License

MIT — see [LICENSE](LICENSE).
