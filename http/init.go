package http

import (
	"github.com/esimov/ascii-fluid/wasm/websocket"
)

var ws websocket.HttpParams

func InitServer() {
	ws = websocket.HttpParams{
		Address: "localhost:5000",
		Prefix:  "/",
		Root:    ".",
	}
	ws.Init()
}

func GetParams() websocket.HttpParams {
	return ws
}
