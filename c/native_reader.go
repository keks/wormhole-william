// +build cgo

package main

import (
	"fmt"
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
	ctx          unsafe.Pointer
	buffer       *C.uint8_t
	bufferLength int
	read         C.readf
	seek         C.seekf
}

func NewNativeReader(ctx unsafe.Pointer, read C.readf, seek C.seekf) ReadSeekCloser {
	return native_reader{
		ctx:          ctx,
		buffer:       (*C.uint8_t)(C.malloc(MAX_READ_BUFFER_LEN)),
		bufferLength: MAX_READ_BUFFER_LEN,
		read:         read,
		seek:         seek,
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
	result := C.call_read(r.ctx, r.read, r.buffer, C.int(l))

	if result <= 0 {
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
	result := C.call_seek(r.ctx, r.seek, C.int64_t(offset), C.int(whence))

	if result < 0 {
		return 0, fmt.Errorf("Invalid offset")
	}
	return int64(result), nil
}
