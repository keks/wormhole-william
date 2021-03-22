// +build js,wasm

package wasm

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"syscall/js"
	"unsafe"

	"github.com/psanford/wormhole-william/wormhole"
)

type ClientMap = map[uintptr]*wormhole.Client

// TODO: automate use of `-ld -X` with env vars
const DEFAULT_APP_ID = "myFileTransfer"
const DEFAULT_RENDEZVOUS_URL = "http://localhost:4000/v1"
const DEFAULT_TRANSIT_RELAY_ADDRESS = "ws://localhost:4002"
const DEFAULT_PASSPHRASE_COMPONENT_LENGTH = 2

var (
	ErrClientNotFound = fmt.Errorf("%s", "wormhole client not found")

	clientMap ClientMap
)

func init() {
	clientMap = make(ClientMap)
}

func NewClient(_ js.Value, args []js.Value) interface{} {
	var (
		config js.Value
		object = js.Global().Get("Object")
	)
	if len(args) > 0 && args[0].InstanceOf(object) {
		config = args[0]
	} else {
		config = object.New()
	}

	// read from config
	appID := config.Get("appID")
	rendezvousURL := config.Get("rendezvousURL")
	transitRelayAddress := config.Get("transitRelayAddress")
	passPhraseComponentLength := config.Get("passPhraseComponentLength")

	//overwrite config with defaults where falsy
	//TODO: use constants for property names?
	if !appID.Truthy() {
		config.Set("appID", DEFAULT_APP_ID)
	}
	if !rendezvousURL.Truthy() {
		config.Set("rendezvousURL", DEFAULT_RENDEZVOUS_URL)
	}
	if !transitRelayAddress.Truthy() {
		config.Set("transitRelayAddress", DEFAULT_TRANSIT_RELAY_ADDRESS)
	}
	if !passPhraseComponentLength.Truthy() {
		config.Set("passPhraseComponentLength", DEFAULT_PASSPHRASE_COMPONENT_LENGTH)
	}

	// read config with defaults merged
	// TODO: need this?
	appID = config.Get("appID")
	rendezvousURL = config.Get("rendezvousURL")
	transitRelayAddress = config.Get("transitRelayAddress")
	passPhraseComponentLength = config.Get("passPhraseComponentLength")

	client := &wormhole.Client{
		AppID:                     appID.String(),
		RendezvousURL:             rendezvousURL.String(),
		TransitRelayURL:           transitRelayAddress.String(),
		PassPhraseComponentLength: passPhraseComponentLength.Int(),
	}
	clientPtr := uintptr(unsafe.Pointer(client))
	clientMap[clientPtr] = client

	return clientPtr
}

func Client_SendText(_ js.Value, args []js.Value) interface{} {
	ctx := context.Background()

	return NewPromise(func(resolve ResolveFn, reject RejectFn) {
		if len(args) != 2 {
			reject(fmt.Errorf("invalid number of arguments: %d. expected: %d", len(args), 2))
			return
		}

		clientPtr := uintptr(args[0].Int())
		msg := args[1].String()
		err, client := getClient(clientPtr)
		if err != nil {
			reject(err)
			return
		}

		code, _, err := client.SendText(ctx, msg)
		if err != nil {
			reject(err)
			return
		}
		resolve(code)
	})
}

func Client_SendFile(_ js.Value, args []js.Value) interface{} {
	ctx := context.Background()

	return NewPromise(func(resolve ResolveFn, reject RejectFn) {
		if len(args) != 3 && len(args) != 4 {
			reject(fmt.Errorf("invalid number of arguments: %d. expected: %s", len(args), "3 or 4"))
			return
		}

		clientPtr := uintptr(args[0].Int())
		fileName := args[1].String()

		uint8Array := args[2]
		size := uint8Array.Get("byteLength").Int()
		fileData := make([]byte, size)
		js.CopyBytesToGo(fileData, uint8Array)
		fileReader := bytes.NewReader(fileData)

		err, client := getClient(clientPtr)
		if err != nil {
			reject(err)
			return
		}

		var code string
		if len(args) == 4 && !args[3].IsUndefined() {
			withProgress := wormhole.WithProgress(func(sentBytes int64, totalBytes int64) {
				args[3].Invoke(sentBytes, totalBytes)
			})
			code, _, err = client.SendFile(ctx, fileName, fileReader, withProgress)
		} else {
			code, _, err = client.SendFile(ctx, fileName, fileReader)
		}
		if err != nil {
			reject(err)
			return
		}
		resolve(code)
	})
}

func Client_RecvText(_ js.Value, args []js.Value) interface{} {
	ctx := context.Background()

	return NewPromise(func(resolve ResolveFn, reject RejectFn) {
		if len(args) != 2 {
			reject(fmt.Errorf("invalid number of arguments: %d. expected: %d", len(args), 2))
			return
		}

		clientPtr := uintptr(args[0].Int())
		code := args[1].String()
		err, client := getClient(clientPtr)
		if err != nil {
			reject(err)
			return
		}

		msg, err := client.Receive(ctx, code)
		if err != nil {
			reject(err)
			return
		}

		msgBytes, err := ioutil.ReadAll(msg)
		if err != nil {
			reject(err)
			return
		}
		resolve(string(msgBytes))
	})
}

func Client_RecvFile(_ js.Value, args []js.Value) interface{} {
	ctx := context.Background()

	return NewPromise(func(resolve ResolveFn, reject RejectFn) {
		if len(args) != 2 {
			reject(fmt.Errorf("invalid number of arguments: %d. expected: %d", len(args), 2))
			return
		}

		clientPtr := uintptr(args[0].Int())
		code := args[1].String()
		err, client := getClient(clientPtr)
		if err != nil {
			reject(err)
			return
		}

		msg, err := client.Receive(ctx, code)
		if err != nil {
			reject(err)
			return
		}

		resolve(NewFileStreamReader(msg))
	})
}

func NewFileStreamReader(msg *wormhole.IncomingMessage) js.Value {
	// TODO: make configurable / parameterized?
	bufSize := 1024 * 4 // 4KiB

	total := 0
	readFunc := func(_ js.Value, args []js.Value) interface{} {
		buf := make([]byte, bufSize)
		return NewPromise(func(resolve ResolveFn, reject RejectFn) {
			if len(args) != 1 {
				reject(fmt.Errorf("invalid number of arguments: %d. expected: %d", len(args), 1))
			}

			jsBuf := args[0]
			_resolve := func(n int, done bool) {
				js.CopyBytesToJS(jsBuf, buf[:n])
				resolve(js.Global().Get("Array").New(n, done))
			}
			n, err := msg.Read(buf)
			total += n
			if err != nil {
				reject(err)
				return
			}
			if msg.ReadDone() {
				_resolve(n, true)
				return
			}
			_resolve(n, false)
		})
	}
	return js.Global().Get("FileStreamReader").New(bufSize, js.FuncOf(readFunc))

	//TODO: refactor JS dependency injection
	// NB: this requires that streamsaver is available at `window.StreamSaver`
}

func Client_free(_ js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return fmt.Errorf("invalid number of arguments: %d. expected: %d", len(args), 2)
	}

	clientPtr := uintptr(args[0].Int())
	delete(clientMap, clientPtr)
	return nil
}

func getClient(clientPtr uintptr) (error, *wormhole.Client) {
	client, ok := clientMap[clientPtr]
	if !ok {
		fmt.Println("clientMap entry missing")
		return ErrClientNotFound, nil
	}

	return nil, client
}
