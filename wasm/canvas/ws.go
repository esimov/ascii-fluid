package canvas

import (
	"syscall/js"

	"github.com/esimov/ascii-fluid/http"
)

func (c *Canvas) InitWebSocket() {
	webSocketParams := http.GetParams()
	c.ws = js.Global().Get("WebSocket").New("ws://" + webSocketParams.Address + "/ws")
	c.Log("Attempting websocket connection...")

	openCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		c.Log("Websocket connection open!")
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

	c.ws.Call("addEventListener", "open", openCallback)
	c.ws.Call("addEventListener", "close", closeCallback)
	c.ws.Call("addEventListener", "error", errorCallback)
}

func (c *Canvas) send(value string) {
	c.ws.Call("send", value)
}
