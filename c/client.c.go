package main

import (
	"github.com/psanford/wormhole-william/c/codes"
	"github.com/psanford/wormhole-william/wormhole"
	"unsafe"
)

import "C"

func main() {

}

// TODO: refactor?
const (
	DEFAULT_APP_ID                      = "myFileTransfer"
	DEFAULT_RENDEZVOUS_URL              = "ws://localhost:4000/v1"
	DEFAULT_TRANSIT_RELAY_URL           = "ws://localhost:4002"
	DEFAULT_PASSPHRASE_COMPONENT_LENGTH = 2
)

var clientsMap = make(map[uintptr]*wormhole.Client)

//export NewClient
func NewClient() uintptr {
	// TODO: receive config
	client := &wormhole.Client{
		AppID: DEFAULT_APP_ID,
		RendezvousURL: DEFAULT_RENDEZVOUS_URL,
		TransitRelayURL: DEFAULT_TRANSIT_RELAY_URL,
		PassPhraseComponentLength: DEFAULT_PASSPHRASE_COMPONENT_LENGTH,
	}

	ptr := uintptr(unsafe.Pointer(client))
	clientsMap[ptr] = client

	return ptr
}

//export FreeClient
func FreeClient(clientPtr uintptr) C.int {
	if _, ok := clientsMap[clientPtr]; !ok {
		return C.int(codes.ERR_NO_CLIENT)
	}
	delete(clientsMap, clientPtr)
	return C.int(codes.OK)
}

// TODO: opts ...TransferOptions
// TODO: r io.ReadSeeker
////export ClientSendText
//func ClientSendText(clientPtr uintptr, fileName string, ) int {
//
//}
