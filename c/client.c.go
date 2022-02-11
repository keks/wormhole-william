// +build cgo

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
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
	DOWNLOAD = 0
	REJECT   = 1
)

type ClientsMap = map[uintptr]*wormhole.Client

var (
	ErrClientNotFound = fmt.Errorf("%s", "wormhole client not found")

	clientsMap       ClientsMap
	pendingDownloads map[int]chan int = map[int]chan int{}
)

func init() {
	clientsMap = make(ClientsMap)
}

func progressHandler(context unsafe.Pointer, progress *C.progress_t, pcb C.progress_cb) wormhole.TransferOption {
	return wormhole.WithProgress(
		func(done int64, total int64) {
			*progress = C.progress_t{
				transferred_bytes: C.int64_t(done),
				total_bytes:       C.int64_t(total),
			}
			C.update_progress(unsafe.Pointer(context), pcb, progress)
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

func codeGenResult(errorCode int, errorString string, code string) *C.codegen_result_t {
	codeGenResultC := (*C.codegen_result_t)(C.malloc(C.sizeof_codegen_result_t))
	*codeGenResultC = C.codegen_result_t{
		error_code:   C.int(errorCode),
		error_string: nil,
		code:         nil,
	}

	if errorString != "" {
		codeGenResultC.error_string = C.CString(errorString)
	}

	if code != "" {
		codeGenResultC.code = C.CString(code)
	}

	return codeGenResultC
}

//export ClientSendText
func ClientSendText(ptrC unsafe.Pointer, clientPtr uintptr, msgC *C.char, cb C.async_cb) *C.codegen_result_t {
	ctx := context.Background()
	client, err := getClient(clientPtr)
	if err != nil {
		return codeGenResult(int(codes.ERR_NO_CLIENT), err.Error(), "")
	}

	// TODO: return code asynchronously (i.e. from a go routine).
	//	This call blocks on network I/O with the mailbox.
	code, status, err := client.SendText(ctx, C.GoString(msgC))
	if err != nil {
		return codeGenResult(int(codes.ERR_SEND_TEXT), err.Error(), "")
	}

	go func() {
		resultC := (*C.result_t)(C.malloc(C.sizeof_result_t))
		*resultC = C.result_t{}
		s := <-status
		if s.Error != nil {
			resultC.err_code = C.int(codes.ERR_SEND_TEXT_RESULT)
			resultC.err_string = C.CString(s.Error.Error())
		} else if s.OK {
			resultC.err_code = C.int(codes.OK)
		} else {
			resultC.err_code = C.int(codes.ERR_UNKNOWN)
			resultC.err_string = C.CString(codes.ERR_UNKNOWN.String())
		}
		C.call_callback(ptrC, cb, resultC)
	}()

	return codeGenResult(int(codes.OK), "", code)
}

//export ClientSendFile
func ClientSendFile(nativeContext unsafe.Pointer, clientPtr uintptr, fileName *C.char,
	cb C.async_cb, pcb C.progress_cb, read C.readf, seek C.seekf) *C.codegen_result_t {
	client, err := getClient(clientPtr)
	if err != nil {
		return codeGenResult(int(codes.ERR_NO_CLIENT), err.Error(), "")
	}
	ctx := context.Background()

	reader := NewNativeReader(nativeContext, read, seek)

	progress := (*C.progress_t)(C.malloc(C.sizeof_progress_t))
	whenComplete := func() {
		C.free(unsafe.Pointer(progress))
		reader.Close()
	}

	// TODO: return code asynchronously (i.e. from a go routine).
	//	This call blocks on network I/O with the mailbox.
	code, status, err := client.SendFile(ctx, C.GoString(fileName), reader, false, progressHandler(nativeContext, progress, pcb))

	if err != nil {
		whenComplete()
		return codeGenResult(int(codes.ERR_SEND_TEXT), err.Error(), "")
	}

	go func() {
		resultC := (*C.result_t)(C.malloc(C.sizeof_result_t))
		defer whenComplete()
		*resultC = C.result_t{}
		s := <-status
		if s.Error != nil {
			resultC.err_code = C.int(codes.ERR_SEND_FILE_RESULT)
			resultC.err_string = C.CString(s.Error.Error())
		} else if s.OK {
			resultC.err_code = C.int(codes.OK)
		} else {
			resultC.err_code = C.int(codes.ERR_UNKNOWN)
			resultC.err_string = C.CString("Unknown error")
		}
		C.call_callback(nativeContext, cb, resultC)
	}()

	return codeGenResult(int(codes.OK), "", code)
}

//export ClientRecvText
func ClientRecvText(ptrC unsafe.Pointer, clientPtr uintptr, codeC *C.char, cb C.async_cb) int {
	client, err := getClient(clientPtr)
	if err != nil {
		return int(codes.ERR_NO_CLIENT)
	}
	ctx := context.Background()

	go func() {
		resultC := (*C.result_t)(C.malloc(C.sizeof_result_t))
		*resultC = C.result_t{}
		msg, err := client.Receive(ctx, C.GoString(codeC), false)
		if err != nil {
			resultC.err_code = C.int(codes.ERR_RECV_TEXT)
			resultC.err_string = C.CString(err.Error())
			C.call_callback(ptrC, cb, resultC)
			return
		}

		data, err := ioutil.ReadAll(msg)
		if err != nil {
			resultC.err_code = C.int(codes.ERR_RECV_TEXT_DATA)
			resultC.err_string = C.CString(err.Error())
			C.call_callback(ptrC, cb, resultC)
			return
		}

		resultC.received_text = C.CString(string(data))
		resultC.err_code = C.int(codes.OK)
		C.call_callback(ptrC, cb, resultC)
	}()

	return int(codes.OK)
}

//export ClientRecvFile
func ClientRecvFile(ptrC unsafe.Pointer, clientPtr uintptr, codeC *C.char,
	cb C.async_cb, pcb C.progress_cb, fmdcb C.file_metadata_cb, write C.writef) int {
	client, err := getClient(clientPtr)
	if err != nil {
		return int(codes.ERR_NO_CLIENT)
	}
	ctx := context.Background()

	resultC := (*C.result_t)(C.malloc(C.sizeof_result_t))
	*resultC = C.result_t{}
	progress := (*C.progress_t)(C.malloc(C.sizeof_progress_t))
	metadata := (*C.file_metadata_t)(C.malloc(C.sizeof_file_metadata_t))

	go func() {
		msg, err := client.Receive(ctx, C.GoString(codeC), false, progressHandler(ptrC, progress, pcb))

		if err != nil {
			resultC.err_code = C.int(codes.ERR_RECV_FILE)
			resultC.err_string = C.CString(err.Error())
			C.call_callback(ptrC, cb, resultC)
			return
		}

		metadata.length = C.int64_t(msg.UncompressedBytes64)
		metadata.file_name = C.CString(msg.Name)
		metadata.context = ptrC
		metadata.download_id = C.int(len(pendingDownloads))

		pendingDownloads[int(metadata.download_id)] = make(chan int)

		fmt.Printf("Calling update metadata")

		C.update_metadata(ptrC, fmdcb, metadata)

		download := func() {
			fmt.Printf("Download function called")
			c_buffer := C.malloc(MAX_READ_BUFFER_LEN)
			defer C.free(c_buffer)
			defer C.free(unsafe.Pointer(progress))

			buffer := make([]byte, MAX_READ_BUFFER_LEN)

			var bytesRead int

			// TODO maybe err == nil || bytesRead > 0 in case of EOF after reading some bytes?
			for bytesRead, err = msg.Read(buffer); bytesRead > 0; bytesRead, err = msg.Read(buffer) {
				for i := 0; i < bytesRead; i++ {
					index := (*C.uint8_t)(unsafe.Pointer(uintptr(unsafe.Pointer(c_buffer)) + uintptr(i)))
					*index = C.uint8_t(buffer[i])
				}

				C.call_write(ptrC, write, (*C.uint8_t)(c_buffer), C.int(bytesRead))
			}

			if err != nil && err != io.EOF {
				resultC.err_code = C.int(codes.ERR_RECV_TEXT_DATA)
				resultC.err_string = C.CString(err.Error())
				C.call_callback(ptrC, cb, resultC)
				return
			}

			resultC.err_code = C.int(codes.OK)
			C.call_callback(ptrC, cb, resultC)
		}

		reject := func() {
			msg.Reject()
			resultC.err_code = C.int(codes.OK)
			C.free(unsafe.Pointer(progress))
			C.call_callback(ptrC, cb, resultC)
		}

		go func() {
			defer func() {
				close(pendingDownloads[int(metadata.download_id)])
				delete(pendingDownloads, int(metadata.download_id))
			}()

			response := <-pendingDownloads[int(metadata.download_id)]
			switch response {
			case DOWNLOAD:
				download()
				break
			case REJECT:
				reject()
				break
			default:
				// TODO proper error code and string
				resultC.err_code = C.int(1337)
				resultC.err_string = C.CString("Invalid response. expecting either DOWNLOAD or REJECT")
				C.call_callback(ptrC, cb, resultC)
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
		fmt.Printf("clientMap entry missing: %d\n", clientPtr)
		return nil, ErrClientNotFound
	}

	return client, nil
}

//export AcceptDownload
func AcceptDownload(downloadId int) {
	pendingDownloads[downloadId] <- DOWNLOAD
}

//export RejectDownload
func RejectDownload(downloadId int) {
	pendingDownloads[downloadId] <- REJECT
}
