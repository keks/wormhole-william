//go:build cgo
// +build cgo

package main

import (
	"io"
	"unsafe"
)

// #include <stdlib.h>
// #include "client.h"
import "C"

const MAX_READ_BUFFER_LEN = 1024 * 64

// TODO this is added in lieu of io.ReadSeekCloser
// which isn't available on older Go versions of the io package
type ReadSeekCloser interface {
	io.ReadSeeker
	Close() error
}

type native_reader struct {
	context      Application
	buffer       *C.uint8_t
	bufferLength int
}

func NewNativeReader(ctx Application) ReadSeekCloser {
	return native_reader{
		context:      ctx,
		buffer:       (*C.uint8_t)(C.malloc(MAX_READ_BUFFER_LEN)),
		bufferLength: MAX_READ_BUFFER_LEN,
	}
}

func (r native_reader) Close() error {
	C.free((unsafe.Pointer)(r.buffer))
	return nil
}

func (r native_reader) Read(buffer []byte) (int, error) {
	l := r.bufferLength
	if len(buffer) < r.bufferLength {
		l = len(buffer)
	}
	result, err := r.context.Read(r.buffer, l)

	if err != nil {
		return 0, err
	} else if result <= 0 {
		return 0, io.EOF
	} else {
		// TODO probably something like memcpy can be done here with unsafe.Slice?
		for i := 0; i < int(result); i++ {
			buffer[i] = *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(r.buffer)) + uintptr(i)))
		}
		return int(result), nil
	}

}

func (r native_reader) Seek(offset int64, whence int) (int64, error) {
	result, err := r.context.Seek(offset, whence)

	if err != nil {
		return -1, err
	}
	return result, nil
}
