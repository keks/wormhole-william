//+build cgo

package main

import (
	"context"
	"fmt"
	"unsafe"
	"log"

	"github.com/psanford/wormhole-william/c/codes"
	"github.com/psanford/wormhole-william/wormhole"
	"github.com/psanford/wormhole-william/rendezvous"
	"github.com/psanford/wormhole-william/internal/crypto"
	"io/ioutil"
)

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
type ContextMap = map[int]context.Context
type RcMap      = map[int]*rendezvous.Client

var (
	ErrClientNotFound = fmt.Errorf("%s", "wormhole client not found")

	clientsMap ClientsMap
	contextMap ContextMap
	rcMap      RcMap
	ctxIndex   int
	rcIndex    int
)

func init() {
	clientsMap = make(ClientsMap)
	contextMap = make(ContextMap)
	rcMap      = make(RcMap)
	ctxIndex = 0
	rcIndex  = 0
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

//export NewContext
func NewContext() C.int {
	ctx := context.Background()
	contextMap[ctxIndex] = ctx

	return C.int(ctxIndex)
}

//export DeleteContext
func DeleteContext(ctxIndex C.int) {
	delete(contextMap, int(ctxIndex))
}

//export FreeClient
func FreeClient(clientPtr uintptr) int {
	if _, err := getClient(clientPtr); err != nil {
		return int(codes.ERR_NO_CLIENT)
	}

	delete(clientsMap, clientPtr)
	return int(codes.OK)
}

//export ClientAppID
func ClientAppID(clientPtr uintptr) *C.char {
	client, err := getClient(clientPtr)
	if err != nil {
		return nil
	}

	appID := client.GetAppID()

	return C.CString(appID)
}

//export ClientSideID
func ClientSideID() *C.char {
	sideID := crypto.RandSideID()

	return C.CString(sideID)
}

// return rendezvous client ptr and code if success or a null ptr
// in case of failure.
//export ClientGetCode
func ClientGetCode(clientPtr uintptr, ctxIndex C.int, sideID *C.char, appID *C.char, codeOutC **C.char) int {
	client, err := getClient(clientPtr)
	if err != nil {
		return int(0)
	}
	ctx := contextMap[int(ctxIndex)]

	code, rc, err := client.CreateOrAttachMailbox(ctx, C.GoString(sideID), C.GoString(appID), "")
	if err != nil {
		return int(-1)
	}

	rcMap[rcIndex] = rc

	*codeOutC = C.CString(code)
	return int(rcIndex)
}


//export ClientSendTextMsg
func ClientSendTextMsg(clientPtr uintptr, ctxIndex C.int, rcIndex uintptr, sideID *C.char, appID *C.char, code *C.char, msg *C.char) int {
	client, err := getClient(clientPtr)
	if err != nil {
		return int(-1)
	}

	ctx := contextMap[int(ctxIndex)]
	rc := rcMap[int(rcIndex)]

	if rc == nil {
		log.Fatal("rc is nil")
	}
	
	status, err := client.SendTextMsg(ctx, rc, C.GoString(sideID), C.GoString(appID), C.GoString(code), C.GoString(msg), nil)
	if err != nil {
		log.Printf("%v\n", err)
		return int(codes.ERR_SEND_TEXT)
	}

	s := <-status
	if s.Error != nil {
		return int(-1)
	} else if s.OK {
		return int(0)
	} else {
		return int(-1)
	}

	return int(codes.OK)
}

//export ClientSendText
func ClientSendText(clientPtr uintptr, ctxIndex C.int, msgC *C.char, codeOutC **C.char) int {
	client, err := getClient(clientPtr)
	if err != nil {
		return int(codes.ERR_NO_CLIENT)
	}
	ctx := contextMap[int(ctxIndex)]

	code, status, err := client.SendText(ctx, C.GoString(msgC))
	if err != nil {
		log.Printf("%v\n", err)
		return int(codes.ERR_SEND_TEXT)
	}
	fmt.Printf("code returned: %s\n", code)
	*codeOutC = C.CString(code)

	s := <-status
	if s.Error != nil {
		return int(-1)
	} else if s.OK {
		return int(0)
	} else {
		return int(-1)
	}

	return int(codes.OK)
}

//export ClientRecvText
func ClientRecvText(clientPtr uintptr, codeC *C.char, msgOutC **C.char) int {
	client, err := getClient(clientPtr)
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
func getClient(clientPtr uintptr) (*wormhole.Client, error) {
	client, ok := clientsMap[clientPtr]
	if !ok {
		fmt.Printf("clientMap entry missing: %d\n", uintptr(clientPtr))
		fmt.Printf("clientMap entry missing: %d\n", clientPtr)
		return nil, ErrClientNotFound
	}

	return client, nil
}
