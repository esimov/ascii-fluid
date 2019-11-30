package http

import (
	"github.com/esimov/ascii-fluid/wasm/websocket"
)

var ws = &websocket.HttpParams{
	Address: "localhost:5000",
	Prefix:  "/",
	Root:    ".",
}

func InitServer() {
	websocket.Init(ws)
}

func GetParams() *websocket.HttpParams {
	return ws
}
