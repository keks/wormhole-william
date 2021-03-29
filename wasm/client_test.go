// +build js,wasm

package wasm

import (
	"context"
	"regexp"
	"syscall/js"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	wasmCtx    context.Context
	wasmCancel context.CancelFunc
)

func TestMain(m *testing.M) {
	wasmCtx, wasmCancel = context.WithCancel(context.Background())
	wasmCtx.Done()

	m.Run()
}

func TestClient_SendText(t *testing.T) {
	ctx, _ := context.WithDeadline(wasmCtx, time.Now().Add(3*time.Second))
	text := "testing 123"

	jsClientPtr := NewClientWrapper()
	require.NotZero(t, jsClientPtr)

	sendPromise := Client_SendText(js.Undefined(), []js.Value{jsClientPtr, js.ValueOf(text)})
	require.NotNil(t, sendPromise)

	_, ok := sendPromise.(*Promise)
	require.True(t, ok)

	resolved := false
	js.ValueOf(sendPromise).Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		resolved = true
		require.Len(t, args, 1)
		require.NotZero(t, args[0])

		codeMatches, err := regexp.MatchString("\\d+(-\\w+)+", args[0].String())
		require.NoError(t, err)
		require.True(t, codeMatches)

		wasmCancel()
		return nil
	}))

	<-ctx.Done()
	require.True(t, resolved)
}

func NewClientWrapper() js.Value {
	return js.FuncOf(NewClient).Invoke(js.Undefined(), nil)
}
