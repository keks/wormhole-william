//+build cgo

package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"unsafe"

	"github.com/psanford/wormhole-william/c/codes"
	"github.com/psanford/wormhole-william/wormhole"
)

import "C"

func main() {

}

// TODO: refactor?
const (
	DEFAULT_APP_ID                      = "myFileTransfer"
	DEFAULT_RENDEZVOUS_URL              = "tcp:transit.magic-wormhole.io:4001"
	DEFAULT_TRANSIT_RELAY_URL           = "ws://localhost:4002"
	DEFAULT_PASSPHRASE_COMPONENT_LENGTH = 2
)

// TODO: figure out how to get uintptr key to work.
type ClientsMap = map[int32]*wormhole.Client

var (
	ErrClientNotFound = fmt.Errorf("%s", "wormhole client not found")

	clientsMap  ClientsMap
	clientIndex int32 = 0
)

func init() {
	clientsMap = make(ClientsMap)
}

//export NewClient
func NewClient() int32 {
	// TODO: receive config
	client := &wormhole.Client{
		AppID:                     DEFAULT_APP_ID,
		RendezvousURL:             DEFAULT_RENDEZVOUS_URL,
		TransitRelayURL:           DEFAULT_TRANSIT_RELAY_URL,
		PassPhraseComponentLength: DEFAULT_PASSPHRASE_COMPONENT_LENGTH,
	}

	fmt.Printf("clientsMap: %+v\n", clientsMap)
	clientIndex++
	clientsMap[clientIndex] = client
	fmt.Printf("clientsMap: %+v\n", clientsMap)

	return clientIndex
}

//export FreeClient
func FreeClient(clientIndex int32) int {
	if _, err := getClient(clientIndex); err != nil {
		return int(codes.ERR_NO_CLIENT)
	}

	delete(clientsMap, clientIndex)
	return int(codes.OK)
}

//export ClientSendText
func ClientSendText(clientIndex int32, msgC *C.char, codeOutC **C.char) int {
	client, err := getClient(clientIndex)
	if err != nil {
		return int(codes.ERR_NO_CLIENT)
	}
	ctx := context.Background()

	code, _, err := client.SendText(ctx, C.GoString(msgC))
	if err != nil {
		return int(codes.ERR_SEND_TEXT)
	}

	*codeOutC = C.CString(code)
	return int(codes.OK)
}

//export ClientSendFile
func ClientSendFile(clientIndex int32, fileName *C.char, length C.int, fileBytes unsafe.Pointer, codeOutC **C.char) int {
	client, err := getClient(clientIndex)
	if err != nil {
		return int(codes.ERR_NO_CLIENT)
	}
	ctx := context.Background()

	reader := bytes.NewReader(C.GoBytes(fileBytes, length))

	code, _, err := client.SendFile(ctx, C.GoString(fileName), reader)
	if err != nil {
		return int(codes.ERR_SEND_TEXT)
	}

	*codeOutC = C.CString(code)
	return int(codes.OK)
}

//export ClientRecvText
func ClientRecvText(clientIndex int32, codeC *C.char, msgOutC **C.char) int {
	client, err := getClient(clientIndex)
	if err != nil {
		return int(codes.ERR_NO_CLIENT)
	}
	ctx := context.Background()

	msg, err := client.Receive(ctx, C.GoString(codeC))
	if err != nil {
		return int(codes.ERR_SEND_TEXT)
	}

	data, err := ioutil.ReadAll(msg)
	if err != nil {
		return int(codes.ERR_RECV_TEXT)
	}

	*msgOutC = C.CString(string(data))
	return int(codes.OK)
}

// TODO: refactor w/ wasm package?
func getClient(clientPtr int32) (*wormhole.Client, error) {
	client, ok := clientsMap[clientPtr]
	if !ok {
		fmt.Printf("clientMap entry missing: %d\n", uintptr(clientPtr))
		fmt.Printf("clientMap entry missing: %d\n", clientPtr)
		return nil, ErrClientNotFound
	}

	return client, nil
}
