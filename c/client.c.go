//+build cgo

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"unsafe"

	"github.com/psanford/wormhole-william/c/codes"
	"github.com/psanford/wormhole-william/wormhole"
	"io/ioutil"
)

// #include "client.h"
import "C"

func main() {

}

// TODO: refactor?
const (
	DEFAULT_APP_ID                      = "lothar.com/wormhole/text-or-file-xfer"
	DEFAULT_RENDEZVOUS_URL              = "ws://relay.magic-wormhole.io:4000/v1"
	DEFAULT_TRANSIT_RELAY_URL           = "tcp:transit.magic-wormhole.io:4001"
	DEFAULT_PASSPHRASE_COMPONENT_LENGTH = 2
)

// TODO: figure out how to get uintptr key to work.
type ClientsMap = map[uintptr]*wormhole.Client

var (
	ErrClientNotFound = fmt.Errorf("%s", "wormhole client not found")

	clientsMap ClientsMap
)

func init() {
	clientsMap = make(ClientsMap)
}

//export NewClient
func NewClient() uintptr {
	// TODO: receive config
	client := &wormhole.Client{
		AppID: DEFAULT_APP_ID,
		RendezvousURL: DEFAULT_RENDEZVOUS_URL,
		TransitRelayURL: DEFAULT_TRANSIT_RELAY_URL,
		PassPhraseComponentLength: DEFAULT_PASSPHRASE_COMPONENT_LENGTH,
	}

	clientPtr := uintptr(unsafe.Pointer(client))
	clientsMap[clientPtr] = client

	return clientPtr
}

//export FreeClient
func FreeClient(clientPtr uintptr) int {
	if _, err := getClient(clientPtr); err != nil {
		return int(codes.ERR_NO_CLIENT)
	}

	delete(clientsMap, clientPtr)
	return int(codes.OK)
}

//export ClientSendText
func ClientSendText(ctxC unsafe.Pointer, clientPtr uintptr, msgC *C.char, codeOutC **C.char, cb C.callback) int {
	client, err := getClient(clientPtr)
	if err != nil {
		return int(codes.ERR_NO_CLIENT)
	}
	ctx := context.Background()

	code, status, err := client.SendText(ctx, C.GoString(msgC))
	if err != nil {
		log.Printf("%v\n", err)
		return int(codes.ERR_SEND_TEXT)
	}
	fmt.Printf("code returned: %s\n", code)
	*codeOutC = C.CString(code)

	go func() {
		s := <-status
		if s.Error != nil {
			// TODO: stick error message somewhere conventional for C to read.
			C.call_callback(ctxC, cb, nil, C.int(codes.ERR_SEND_TEXT_RESULT))
		} else if s.OK {
			C.call_callback(ctxC, cb, nil, C.int(codes.OK))
		} else {
			C.call_callback(ctxC, cb, nil, C.int(codes.ERR_UNKNOWN))
		}
	}()

	return int(codes.OK)
}

//export ClientSendFile
func ClientSendFile(ctxC unsafe.Pointer, clientPtr uintptr, fileName *C.char, length C.int, fileBytes unsafe.Pointer, codeOutC **C.char, cb C.callback) int {
	client, err := getClient(clientPtr)
	if err != nil {
		return int(codes.ERR_NO_CLIENT)
	}
	ctx := context.Background()

	reader := bytes.NewReader(C.GoBytes(fileBytes, length))

	code, status, err := client.SendFile(ctx, C.GoString(fileName), reader)
	if err != nil {
		return int(codes.ERR_SEND_TEXT)
	}
	*codeOutC = C.CString(code)

	go func() {
		s := <-status
		if s.Error != nil {
			// TODO: stick error message somewhere conventional for C to read.
			C.call_callback(ctxC, cb, nil, C.int(codes.ERR_SEND_TEXT_RESULT))
		} else if s.OK {
			C.call_callback(ctxC, cb, nil, C.int(codes.OK))
		} else {
			C.call_callback(ctxC, cb, nil, C.int(codes.ERR_UNKNOWN))
		}
	}()

	return int(codes.OK)
}

//export ClientRecvText
func ClientRecvText(ctxC unsafe.Pointer, clientPtr uintptr, codeC *C.char, cb C.callback) int {
	client, err := getClient(clientPtr)
	if err != nil {
		return int(codes.ERR_NO_CLIENT)
	}
	ctx := context.Background()

	go func() {
		msg, err := client.Receive(ctx, C.GoString(codeC))

		if err != nil {
			C.call_callback(ctxC, cb, nil, C.int(codes.ERR_RECV_TEXT))
		}

		data, err := ioutil.ReadAll(msg)
		if err != nil {
			C.call_callback(ctxC, cb, nil, C.int(codes.ERR_RECV_TEXT_DATA))
		}
		dataC := C.CBytes(data)
		fileC := (*C.file_t)(C.malloc(C.sizeof_file_t))
		*fileC = C.file_t{
		   length: C.int(len(data)),
		   data: (*C.uint8_t)(dataC),
		}
		fmt.Printf("Go | client.c.go:158 fileC: %p/n", fileC);

// 		C.call_callback(ctxC, cb, unsafe.Pointer(C.CString(string(data))), C.int(codes.OK))
		C.call_callback(ctxC, cb, unsafe.Pointer(fileC), C.int(codes.OK))
	}()

	return int(codes.OK)
}

// TODO: refactor w/ wasm package?
func getClient(clientPtr uintptr) (*wormhole.Client, error) {
	client, ok := clientsMap[clientPtr]
	if !ok {
		fmt.Printf("clientMap entry missing: %d\n", uintptr(clientPtr))
		fmt.Printf("clientMap entry missing: %d\n", clientPtr)
		return nil, ErrClientNotFound
	}

	return client, nil
}
