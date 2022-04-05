//go:build cgo
// +build cgo

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
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

const (
	ERR_CONTEXT_CANCELLED = "context canceled"
	ERR_BROKEN_PIPE       = "write: broken pipe"
	ERR_UNEXPECTED_EOF    = "unexpected EOF"
)

type Command int

const (
	DOWNLOAD Command = 0
	REJECT   Command = 1
	CANCEL   Command = 2
)

type ClientsMap = map[uintptr]*wormhole.Client

type transferContext struct {
	Commands   chan Command
	CancelFunc context.CancelFunc
}

var (
	ErrClientNotFound = fmt.Errorf("%s", "wormhole client not found")

	clientsMap       ClientsMap
	pendingTransfers map[int]transferContext = map[int]transferContext{}
)

func init() {
	clientsMap = make(ClientsMap)
}

func progressHandler(context C.client_context_t, progress *C.progress_t, pcb C.update_progressf) wormhole.TransferOption {
	return wormhole.WithProgress(
		func(done int64, total int64) {
			*progress = C.progress_t{
				transferred_bytes: C.int64_t(done),
				total_bytes:       C.int64_t(total),
			}
			C.call_update_progress(context, pcb, progress)
		})
}

//export NewClient
func NewClient(appId *C.char, rendezvousUrl *C.char, transitRelayUrl *C.char, passPhraseComponentLength C.int) uintptr {
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

// TODO when the original error type contains more information than
// the error message, refactor this
func extractErrorCode(fallback C.result_type_t, errorMessage string) C.result_type_t {
	if fallback == C.SendFileError {
		if strings.Contains(errorMessage, ERR_BROKEN_PIPE) ||
			strings.Contains(errorMessage, ERR_CONTEXT_CANCELLED) {
			return C.TransferCancelled
		}
	} else if fallback == C.ReceiveFileError {
		if strings.Contains(errorMessage, ERR_UNEXPECTED_EOF) ||
			strings.Contains(errorMessage, ERR_CONTEXT_CANCELLED) {
			return C.TransferCancelled
		}
	}
	return fallback
}

func codeGenSuccessful(clientCtx C.client_context_t, transferId int, code string) *C.codegen_result_t {
	result := (*C.codegen_result_t)(C.calloc(1, C.sizeof_codegen_result_t))
	result.result_type = C.CodeGenSuccessful
	result.generated.code = C.CString(code)
	result.generated.transfer_id = C.int32_t(transferId)
	result.context = clientCtx
	return result
}

func codeGenFailed(clientCtx C.client_context_t, resultType C.codegen_result_type_t, errorMessage string) *C.codegen_result_t {
	result := (*C.codegen_result_t)(C.calloc(1, C.sizeof_codegen_result_t))
	result.result_type = resultType
	result.error.error_string = C.CString(errorMessage)
	return result
}

func failWith(clientCtx C.client_context_t, notify C.notify_resultf, result C.result_type_t, errorMessage string) {
	resultC := (*C.result_t)(C.calloc(1, C.sizeof_result_t))
	resultC.result_type = extractErrorCode(result, errorMessage)
	resultC.err_string = C.CString(errorMessage)
	C.call_notify_result(clientCtx, notify, resultC)
}

func textReceived(clientCtx C.client_context_t, notify C.notify_resultf, text string) {
	resultC := (*C.result_t)(C.calloc(1, C.sizeof_result_t))
	resultC.result_type = C.Success
	resultC.received_text = C.CString(text)
	C.call_notify_result(clientCtx, notify, resultC)
}

func transferSuccessful(clientCtx C.client_context_t, notify C.notify_resultf) {
	resultC := (*C.result_t)(C.calloc(1, C.sizeof_result_t))
	resultC.result_type = C.Success
	C.call_notify_result(clientCtx, notify, resultC)
}

//export ClientSendText
func ClientSendText(clientCtx C.client_context_t, clientPtr uintptr, msgC *C.char, cb C.notify_resultf) *C.codegen_result_t {
	ctx := context.Background()
	client, err := getClient(clientPtr)
	if err != nil {
		return codeGenFailed(clientCtx, C.FailedToGetClient, fmt.Sprintf("Failed to get client with id:%d", clientPtr))
	}

	// TODO: return code asynchronously (i.e. from a go routine).
	//	This call blocks on network I/O with the mailbox.
	code, status, err := client.SendText(ctx, C.GoString(msgC))
	if err != nil {
		return codeGenFailed(clientCtx, C.CodeGenerationFailed, err.Error())
	}

	go func() {
		s := <-status
		if s.Error != nil {
			failWith(clientCtx, cb, C.SendTextError, s.Error.Error())
		} else if s.OK {
			transferSuccessful(clientCtx, cb)
		} else {
			failWith(clientCtx, cb, C.SendTextError, "Unknown error")
		}
	}()

	return codeGenSuccessful(clientCtx, len(pendingTransfers), code)
}

//export ClientSendFile
func ClientSendFile(clientCtx C.client_context_t, clientPtr uintptr, fileName *C.char, cb C.notify_resultf, pcb C.update_progressf,
	read C.readf, seek C.seekf) *C.codegen_result_t {
	client, err := getClient(clientPtr)
	if err != nil {
		return codeGenFailed(clientCtx, C.FailedToGetClient, fmt.Sprintf("Failed to get client with id:%d", clientPtr))
	}
	ctx, cancel := context.WithCancel(context.Background())

	reader := NewNativeReader(clientCtx, read, seek)

	progress := (*C.progress_t)(C.malloc(C.sizeof_progress_t))

	transferCtx := transferContext{
		Commands:   make(chan Command),
		CancelFunc: cancel,
	}

	transferId := len(pendingTransfers)
	pendingTransfers[transferId] = transferCtx

	whenComplete := func() {
		C.free(unsafe.Pointer(progress))
		delete(pendingTransfers, transferId)
		reader.Close()
	}

	// TODO: return code asynchronously (i.e. from a go routine).
	//	This call blocks on network I/O with the mailbox.
	code, status, err := client.SendFile(ctx, C.GoString(fileName), reader, true, progressHandler(clientCtx, progress, pcb))

	if err != nil {
		whenComplete()
		return codeGenFailed(clientCtx, C.CodeGenerationFailed, fmt.Sprintf("Failed to generate code for client.SendFile: %v", err))
	}

	go func() {
		for msg := range pendingTransfers[transferId].Commands {
			if msg == CANCEL {
				transferCtx.CancelFunc()
				break
			}
		}
	}()

	go func() {
		defer whenComplete()
		s := <-status
		if s.Error != nil {
			failWith(clientCtx, cb, C.SendFileError, s.Error.Error())
		} else if s.OK {
			transferSuccessful(clientCtx, cb)
		} else {
			failWith(clientCtx, cb, C.SendFileError, "Unknown error")
		}
	}()

	return codeGenSuccessful(clientCtx, transferId, code)
}

//export ClientRecvText
func ClientRecvText(clientCtx C.client_context_t, clientPtr uintptr, codeC *C.char, cb C.notify_resultf) int {
	client, err := getClient(clientPtr)
	if err != nil {
		return int(codes.ERR_NO_CLIENT)
	}
	ctx := context.Background()

	go func() {
		msg, err := client.Receive(ctx, C.GoString(codeC), false)
		if err != nil {
			failWith(clientCtx, cb, C.ReceiveTextError, err.Error())
			return
		}

		data, err := ioutil.ReadAll(msg)
		if err != nil {
			failWith(clientCtx, cb, C.ReceiveTextError, err.Error())
			return
		}

		textReceived(clientCtx, cb, string(data))
	}()

	return int(codes.OK)
}

//export ClientRecvFile
func ClientRecvFile(clientCtx C.client_context_t, clientPtr uintptr, codeC *C.char, cb C.notify_resultf, pcb C.update_progressf,
	umdf C.update_metadataf, write C.writef) int {
	client, err := getClient(clientPtr)
	if err != nil {
		return int(codes.ERR_NO_CLIENT)
	}
	ctx, cancelFunc := context.WithCancel(context.Background())

	go func() {
		progress := (*C.progress_t)(C.malloc(C.sizeof_progress_t))
		metadata := (*C.file_metadata_t)(C.malloc(C.sizeof_file_metadata_t))

		msg, err := client.Receive(ctx, C.GoString(codeC), true, progressHandler(clientCtx, progress, pcb))

		if err != nil {
			failWith(clientCtx, cb, C.ReceiveFileError, err.Error())
			return
		}

		metadata.length = C.int64_t(msg.UncompressedBytes64)
		metadata.file_name = C.CString(msg.Name)
		metadata.context = clientCtx
		metadata.download_id = C.int(len(pendingTransfers))

		pendingTransfers[int(metadata.download_id)] = transferContext{
			Commands:   make(chan Command),
			CancelFunc: cancelFunc,
		}

		C.call_update_metadata(clientCtx, umdf, metadata)

		download := func() {
			c_buffer := C.malloc(MAX_READ_BUFFER_LEN)
			defer C.free(c_buffer)
			defer C.free(unsafe.Pointer(progress))
			defer func() {
				delete(pendingTransfers, int(metadata.download_id))
			}()

			buffer := make([]byte, MAX_READ_BUFFER_LEN)

			var bytesRead int

			for bytesRead, err = msg.Read(buffer); bytesRead > 0; bytesRead, err = msg.Read(buffer) {
				for i := 0; i < bytesRead; i++ {
					index := (*C.uint8_t)(unsafe.Pointer(uintptr(unsafe.Pointer(c_buffer)) + uintptr(i)))
					*index = C.uint8_t(buffer[i])
				}

				completed := int(C.call_write(clientCtx, write, (*C.uint8_t)(c_buffer), C.int(bytesRead))) != 0
				if !completed {
					err = fmt.Errorf("Failed to write to file")
					break
				}
			}

			if err != nil && err != io.EOF {
				failWith(clientCtx, cb, C.ReceiveFileError, err.Error())
				return
			}

			transferSuccessful(clientCtx, cb)
		}

		reject := func() {
			msg.Reject()
			failWith(clientCtx, cb, C.TransferRejected, "Transfer rejected")
			C.free(unsafe.Pointer(progress))
		}

		go func() {
			for response := range pendingTransfers[int(metadata.download_id)].Commands {
				switch response {
				case DOWNLOAD:
					go download()
				case REJECT:
					reject()
					delete(pendingTransfers, int(metadata.download_id))
				case CANCEL:
					pendingTransfers[int(metadata.download_id)].CancelFunc()
					delete(pendingTransfers, int(metadata.download_id))
				}
			}
		}()

	}()

	return int(codes.OK)
}

// TODO: refactor w/ wasm package?
func getClient(clientPtr uintptr) (*wormhole.Client, error) {
	client, ok := clientsMap[clientPtr]
	if !ok {
		fmt.Printf("clientMap entry missing: %d\n", uintptr(clientPtr))
		return nil, ErrClientNotFound
	}

	return client, nil
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
