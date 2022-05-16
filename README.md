# wormhole-william

wormhole-william is a Go (golang) implementation of [magic wormhole](https://magic-wormhole.readthedocs.io/en/latest/). It provides secure end-to-end encrypted file transfers between computers. The endpoints are connected using the same "wormhole code". A secure session is established via SPAKE2 cryptography and uses the [gospake2](https://salsa.debian.org/vasudev/gospake2), an implementation of SPAKE2 algorithm as [implementated](https://github.com/warner/python-spake2) and used by the Python [magic-wormhole](https://github.com/magic-wormhole/magic-wormhole) client. 

wormhole-william is compatible with the official [python magic wormhole cli tool](https://github.com/warner/magic-wormhole).

Currently, wormhole-william supports:
- sending and receiving text over the wormhole protocol
- sending and receiving files over the transit protocol
- sending and receiving directories over the transit protocol

## Docs

https://pkg.go.dev/github.com/psanford/wormhole-william/wormhole?tab=doc

The magic-wormhole protocol documents are described in the [magic-wormhole-protocols](https://github.com/magic-wormhole/magic-wormhole-protocols) repository.

## CLI Usage

```
$ wormhole-william send --help
Send a text message, file, or directory...

Usage:
  wormhole-william send [WHAT] [flags]

Flags:
      --code string       human-generated code phrase
  -c, --code-length int   length of code (in bytes/words)
  -h, --help              help for send
      --hide-progress     suppress progress-bar display
      --text string       text message to send, instead of a file.
                          Use '-' to read from stdin
  -v, --verify            display verification string (and wait for approval)

Global Flags:
      --appid string            AppID to use (default "lothar.com/wormhole/text-or-file-xfer")
      --no-listen               (debug) don't open a listening socket for transit
      --relay-url string        rendezvous relay to use (default "ws://relay.magic-wormhole.io:4000/v1")
      --transit-helper string   relay server url (default "tcp:transit.magic-wormhole.io:4001")


$ wormhole-william receive --help
Receive a text message, file, or directory...

Usage:
  wormhole-william receive [OPTIONS] [CODE]... [flags]

Aliases:
  receive, recv

Flags:
  -h, --help            help for receive
      --hide-progress   suppress progress-bar display
  -v, --verify          display verification string (and wait for approval)

Global Flags:
      --appid string            AppID to use (default "lothar.com/wormhole/text-or-file-xfer")
      --no-listen               (debug) don't open a listening socket for transit
      --relay-url string        rendezvous relay to use (default "ws://relay.magic-wormhole.io:4000/v1")
      --transit-helper string   relay server url (default "tcp:transit.magic-wormhole.io:4001")

```

### CLI tab completion

The wormhole-william CLI supports shell completion, including completing the receive code.
To enable shell completion follow the instructions from `wormhole-william shell-completion -h`.


## Building the CLI tool

wormhole-william uses go modules so it requires a version of the go tool chain >= 1.11. If you are using a version of go that supports modules you can clone the repo outside of your GOPATH and do a `go build` in the top level directory.

To just install via the go tool run:

```
go install github.com/psanford/wormhole-william@latest
```

## API Usage

Sending text:

```go
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/psanford/wormhole-william/wormhole"
)

func sendText() {
	var c wormhole.Client

	msg := "Dillinger-entertainer"

	ctx := context.Background()

	code, status, err := c.SendText(ctx, msg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("On the other computer, please run: wormhole receive")
	fmt.Printf("Wormhole code is: %s\n", code)

	s := <-status

	if s.OK {
		fmt.Println("OK!")
	} else {
		log.Fatalf("Send error: %s", s.Error)
	}
}

func recvText(code string) {
	var c wormhole.Client

	ctx := context.Background()
	msg, err := c.Receive(ctx, code, false)
	if err != nil {
		log.Fatal(err)
	}

	if msg.Type != wormhole.TransferText {
		log.Fatalf("Expected a text message but got type %s", msg.Type)
	}

	msgBody, err := ioutil.ReadAll(msg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("got message:")
	fmt.Println(msgBody)
}
```

Similarly there are APIs to send file and directories: `SendFile()`
and `SendDirectory()`. Please look at `wormhole/send.go and
wormhole/recv.go` to look at the definitions of these functions.

See the [cli tool](https://github.com/psanford/wormhole-william/tree/master/cmd) and [examples](https://github.com/psanford/wormhole-william/tree/master/examples) directory for working examples of how to use the API to send and receive text, files and directories.

## Third Party Users of Wormhole William

- [wormhole-gui](https://github.com/Jacalz/wormhole-gui): A Magic Wormhole graphical user interface
- [wormhole-william-mobile](https://github.com/psanford/wormhole-william-mobile): Android wormhole-william app
