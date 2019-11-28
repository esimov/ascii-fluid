package main

import (
	"github.com/esimov/ascii-fluid/terminal"
	"github.com/esimov/ascii-fluid/wasm/websocket"
)

func main() {
	term := terminal.New()
	term.Init().Render()

	websocket.Init()
}
