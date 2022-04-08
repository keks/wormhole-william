//go:build cgo
// +build cgo

package main

import (
	"fmt"
	"strings"
	"unsafe"
)

// #include <stdlib.h>
// #include "client.h"
import "C"

const (
	ERR_CONTEXT_CANCELLED = "context canceled"
	ERR_BROKEN_PIPE       = "write: broken pipe"
	ERR_UNEXPECTED_EOF    = "unexpected EOF"
	ERR_TRANSFER_REJECTED = "transfer rejected"
)

type Application interface {
	Log(message string, args ...interface{})
	UpdateProgress(done int64, total int64)
	NotifyError(result C.result_type_t, errorMessage string)
	UpdateMetadata(fileName string, length int64, downloadId int)
	Write(bytes unsafe.Pointer, length int) error
	Read(buffer *C.uint8_t, length int) (int, error)
	Seek(offset int64, whence int) (int64, error)
	NotifySuccess()
	TextReceived(text string)
	ClientId() int32
	InternalContext() C.client_context_t
	Finalize()
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

	if strings.Contains(errorMessage, ERR_TRANSFER_REJECTED) {
		return C.TransferRejected
	}

	return fallback
}

func (wctx *C.wrapped_context_t) Log(message string, args ...interface{}) {
	messageC := C.CString(fmt.Sprintf(message, args...))
	C.call_log(wctx, messageC)
	C.free(unsafe.Pointer(messageC))
}

func (wctx *C.wrapped_context_t) UpdateProgress(done int64, total int64) {
	wctx.progress.transferred_bytes = C.int64_t(done)
	wctx.progress.total_bytes = C.int64_t(total)
	C.call_update_progress(wctx)
}

func (wctx *C.wrapped_context_t) NotifyError(result C.result_type_t, errorMessage string) {
	wctx.Log("Error: ErrorCode:%d %s", int(result), errorMessage)
	wctx.result.result_type = extractErrorCode(result, errorMessage)
	wctx.result.err_string = C.CString(errorMessage)
	C.call_notify(wctx)
}

func (wctx *C.wrapped_context_t) UpdateMetadata(fileName string, length int64, downloadId int) {
	wctx.metadata.length = C.int64_t(length)
	wctx.metadata.file_name = C.CString(fileName)
	wctx.metadata.context = wctx.clientCtx
	wctx.metadata.download_id = C.int32_t(downloadId)
	C.call_update_metadata(wctx)
}

func (wctx *C.wrapped_context_t) Write(bytes unsafe.Pointer, length int) error {
	successful := C.call_write(wctx, (*C.uint8_t)(bytes), C.int32_t(length))
	if !successful {
		return fmt.Errorf("Failed to write to file")
	}

	return nil
}

func (wctx *C.wrapped_context_t) NotifySuccess() {
	wctx.result = C.result_t{
		result_type: C.Success,
	}
	C.call_notify(wctx)
}

func (wctx *C.wrapped_context_t) TextReceived(text string) {
	wctx.result = C.result_t{
		result_type:   C.Success,
		received_text: C.CString(text),
	}
	C.call_notify(wctx)
}

func (wctx *C.wrapped_context_t) Read(buffer *C.uint8_t, length int) (int, error) {
	return int(C.call_read(wctx, buffer, C.int(length))), nil
}

func (wctx *C.wrapped_context_t) Seek(offset int64, whence int) (int64, error) {
	return int64(C.call_seek(wctx, C.int64_t(offset), C.int(whence))), nil
}

func (wctx *C.wrapped_context_t) ClientId() int32 {
	return int32(wctx.go_client_id)
}

func (wctx *C.wrapped_context_t) Finalize() {
	wctx.Log("Finalizing context at %d", int(uintptr(unsafe.Pointer(wctx))))
	C.free_wrapped_context(wctx)
}

func (wctx *C.wrapped_context_t) InternalContext() C.client_context_t {
	return wctx.clientCtx
}
