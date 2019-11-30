package canvas

import (
	"syscall/js"

	"github.com/esimov/ascii-fluid/http"
)

type Socket struct {
	Canvas
}

func (c *Socket) InitWebSocket() {
	webSocketParams := http.GetParams()
	ws := js.Global().Get("WebSocket").New("ws://" + webSocketParams.Address + "/ws")
	c.Log("Attempting websocket connection...")

	openCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		c.Log("Websocket connection open!")
		ws.Call("send", "Hi From the Client!")
		return nil
	})

	closeCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		c.Log("Websocket connection closed: ", event)
		return nil
	})

	errorCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		error := args[0]
		c.Log("Websocket error:", error)
		return nil
	})

	ws.Call("addEventListener", "open", openCallback)
	ws.Call("addEventListener", "close", closeCallback)
	ws.Call("addEventListener", "error", errorCallback)
}
