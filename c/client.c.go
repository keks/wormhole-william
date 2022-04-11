//go:build cgo
// +build cgo

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"unsafe"

	"github.com/psanford/wormhole-william/c/codes"
	"github.com/psanford/wormhole-william/wormhole"
)

// #include <stdlib.h>
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

type Command int

const (
	DOWNLOAD Command = 0
	REJECT   Command = 1
	CANCEL   Command = 2
)

type transferContext struct {
	Commands   chan Command
	CancelFunc context.CancelFunc
}

type clientWithContext struct {
	appContext Application
	client     *wormhole.Client
}

var (
	ErrClientNotFound = fmt.Errorf("%s", "wormhole client not found")

	clientsMap       map[int32]clientWithContext = map[int32]clientWithContext{}
	pendingTransfers map[int]transferContext     = map[int]transferContext{}
	transfersLock    sync.Mutex
)

func removePendingTransfer(transferId int) {
	transfersLock.Lock()
	defer transfersLock.Unlock()

	close(pendingTransfers[transferId].Commands)
	delete(pendingTransfers, transferId)
}

func addPendingTransfer(cancelFunc context.CancelFunc) int {
	transfersLock.Lock()
	defer transfersLock.Unlock()

	transferId := len(pendingTransfers)
	pendingTransfers[transferId] = transferContext{
		Commands:   make(chan Command),
		CancelFunc: cancelFunc,
	}

	return transferId
}

//export NewClient
func NewClient(appId *C.char, rendezvousUrl *C.char, transitRelayUrl *C.char, passPhraseComponentLength C.int) int32 {
	client := &wormhole.Client{
		AppID:                     DEFAULT_APP_ID,
		RendezvousURL:             DEFAULT_RENDEZVOUS_URL,
		TransitRelayURL:           DEFAULT_TRANSIT_RELAY_URL,
		PassPhraseComponentLength: DEFAULT_PASSPHRASE_COMPONENT_LENGTH,
	}

	if appId != nil {
		client.AppID = C.GoString(appId)
	}

	if rendezvousUrl != nil {
		client.RendezvousURL = C.GoString(rendezvousUrl)
	}

	if transitRelayUrl != nil {
		client.TransitRelayURL = C.GoString(transitRelayUrl)
	}

	if passPhraseComponentLength == 0 {
		client.PassPhraseComponentLength = int(passPhraseComponentLength)
	}

	clientId := int32(uintptr(unsafe.Pointer(client)))
	clientsMap[clientId] = clientWithContext{

		client: client,
	}

	return clientId
}

//export Finalize
func Finalize(clientId int32) int32 {
	client, ok := clientsMap[clientId]

	client.appContext.Log("Finalizing client with id %d", clientId)

	if !ok {
		return int32(codes.ERR_NO_CLIENT)
	}

	client.appContext.Finalize()

	delete(clientsMap, clientId)
	return int32(codes.OK)
}

func codeGenSuccessful(wctx Application, transferId int, code string) *C.codegen_result_t {
	result := (*C.codegen_result_t)(C.calloc(1, C.sizeof_codegen_result_t))
	result.result_type = C.CodeGenSuccessful
	result.generated.code = C.CString(code)
	result.generated.transfer_id = C.int32_t(transferId)
	result.context = wctx.InternalContext()
	return result
}

func codeGenFailed(wctx Application, resultType C.codegen_result_type_t, errorMessage string) *C.codegen_result_t {
	result := (*C.codegen_result_t)(C.calloc(1, C.sizeof_codegen_result_t))
	result.result_type = resultType
	result.error.error_string = C.CString(errorMessage)
	result.context = wctx.InternalContext()
	return result
}

//export ClientSendText
func ClientSendText(clientCtx *C.wrapped_context_t, msgC *C.char) *C.codegen_result_t {
	ctx := context.Background()
	client, err := getClientWithContext(clientCtx)
	if err != nil {
		return codeGenFailed(clientCtx, C.FailedToGetClient, fmt.Sprintf("Failed to get client with id:%d", clientCtx.ClientId()))
	}

	// TODO: return code asynchronously (i.e. from a go routine).
	//	This call blocks on network I/O with the mailbox.
	code, status, err := client.client.SendText(ctx, C.GoString(msgC))
	if err != nil {
		return codeGenFailed(clientCtx, C.CodeGenerationFailed, err.Error())
	}

	go func() {
		s := <-status
		if s.Error != nil {
			clientCtx.NotifyError(C.SendTextError, s.Error.Error())
		} else if s.OK {
			clientCtx.NotifySuccess()
		} else {
			clientCtx.NotifyError(C.SendTextError, "Unknown error")
		}
	}()

	return codeGenSuccessful(clientCtx, len(pendingTransfers), code)
}

func sendFile(app Application, fileName string) {
	client, err := getClientWithContext(app)
	if err != nil {
		app.NotifyCodeGenerationFailure(C.FailedToGetClient, err.Error())
		return
	}
	ctx, cancel := context.WithCancel(context.Background())

	reader := NewNativeReader(app)

	transferId := addPendingTransfer(cancel)

	code, status, err := client.client.SendFile(ctx, fileName, reader, true, wormhole.WithProgress(app.UpdateProgress))

	if err != nil {
		app.NotifyCodeGenerationFailure(C.CodeGenerationFailed, err.Error())
	}

	app.NotifyCodeGenerated(transferId, code)

	go func() {
		for msg := range pendingTransfers[transferId].Commands {
			switch msg {
			case CANCEL:
				pendingTransfers[transferId].CancelFunc()
				// TODO this is sent because the current implementation of the client
				// does not put an error when the context is cancelled before the
				// transfer has started
				// This can be removed if/when the client implements that behaviour
				status <- wormhole.SendResult{
					Error: fmt.Errorf(ERR_CONTEXT_CANCELLED),
				}
			}
		}
	}()

	go func() {
		defer removePendingTransfer(transferId)
		s := <-status
		if s.Error != nil {
			app.NotifyError(C.SendFileError, s.Error.Error())
		} else if s.OK {
			app.NotifySuccess()
		} else {
			app.NotifyError(C.SendFileError, "Unknown error")
		}
	}()
}

//export ClientSendFile
func ClientSendFile(clientCtx *C.wrapped_context_t, fileName *C.char) {
	go sendFile(clientCtx, C.GoString(fileName))
}

//export ClientRecvText
func ClientRecvText(clientCtx *C.wrapped_context_t, codeC *C.char) int {
	client, err := getClientWithContext(clientCtx)
	if err != nil {
		return int(codes.ERR_NO_CLIENT)
	}
	ctx := context.Background()

	go func() {
		msg, err := client.client.Receive(ctx, C.GoString(codeC), false)
		if err != nil {
			clientCtx.NotifyError(C.ReceiveTextError, err.Error())
			return
		}

		data, err := ioutil.ReadAll(msg)
		if err != nil {
			clientCtx.NotifyError(C.ReceiveTextError, err.Error())
			return
		}

		clientCtx.TextReceived(string(data))
	}()

	return int(codes.OK)
}

func recvFile(app Application, code string) {
	client, err := getClientWithContext(app)
	if err != nil {
		app.NotifyError(C.ReceiveFileError, err.Error())
		return
	}
	ctx, cancelFunc := context.WithCancel(context.Background())

	msg, err := client.client.Receive(ctx, code, true, wormhole.WithProgress(app.UpdateProgress))

	if err != nil {
		app.NotifyError(C.ReceiveFileError, err.Error())
		return
	}

	downloadId := addPendingTransfer(cancelFunc)

	app.UpdateMetadata(msg.Name, msg.UncompressedBytes64, downloadId)

	download := func() {
		c_buffer := C.malloc(MAX_READ_BUFFER_LEN)
		defer C.free(c_buffer)

		buffer := make([]byte, MAX_READ_BUFFER_LEN)

		var bytesRead int

		for bytesRead, err = msg.Read(buffer); bytesRead > 0 && err == nil; bytesRead, err = msg.Read(buffer) {
			for i := 0; i < bytesRead; i++ {
				index := (*C.uint8_t)(unsafe.Pointer(uintptr(unsafe.Pointer(c_buffer)) + uintptr(i)))
				*index = C.uint8_t(buffer[i])
			}

			if err = app.Write(c_buffer, bytesRead); err != nil {
				break
			}
		}

		if err != nil && err != io.EOF {
			app.NotifyError(C.ReceiveFileError, err.Error())
			return
		}

		app.NotifySuccess()
		removePendingTransfer(downloadId)
	}

	reject := func() {
		msg.Reject()
		app.NotifyError(C.TransferRejected, "Transfer rejected")
		removePendingTransfer(downloadId)
	}

	go func() {
		for response := range pendingTransfers[downloadId].Commands {
			switch response {
			case DOWNLOAD:
				go download()
			case REJECT:
				reject()
			case CANCEL:
				pendingTransfers[downloadId].CancelFunc()
				removePendingTransfer(downloadId)
			}
		}
	}()
}

//export ClientRecvFile
func ClientRecvFile(clientCtx *C.wrapped_context_t, codeC *C.char) {
	go recvFile(clientCtx, C.GoString(codeC))
}

// TODO: refactor w/ wasm package?
func getClientWithContext(clientCtx Application) (*clientWithContext, error) {
	client, ok := clientsMap[clientCtx.ClientId()]
	if !ok {
		clientCtx.Log(fmt.Sprintf("clientMap entry missing: %d\n", clientCtx.ClientId()))
		return nil, ErrClientNotFound
	}

	if client.appContext != nil {
		return nil, fmt.Errorf("Context for client with id %d is already assigned", clientCtx.ClientId())
	}

	client.appContext = clientCtx
	clientsMap[clientCtx.ClientId()] = client

	return &client, nil
}

//export AcceptDownload
func AcceptDownload(transferId int) {
	pendingTransfers[transferId].Commands <- DOWNLOAD
}

//export RejectDownload
func RejectDownload(transferId int) {
	pendingTransfers[transferId].Commands <- REJECT
}

//export CancelTransfer
func CancelTransfer(transferId int) {
	pendingTransfers[transferId].Commands <- CANCEL
}
