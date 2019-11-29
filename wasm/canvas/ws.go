// let socket = new WebSocket("ws://localhost:5000/ws");
// console.log("Attempting Connection...");

// socket.onopen = () => {
// 	console.log("Successfully Connected");
// 	socket.send("Hi From the Client!")
// };

// socket.onclose = event => {
// 	console.log("Socket Closed Connection: ", event);
// 	socket.send("Client Closed!")
// };

// socket.onerror = error => {
// 	console.log("Socket Error: ", error);
// };

package canvas

import (
	"fmt"
	"syscall/js"

	"github.com/esimov/ascii-fluid/http"
)

type Socket struct {
	Canvas
}

func InitWebSocket() {
	ws := http.GetParams()
	socket := js.Global().Get("WebSocket").New("ws://" + ws.Address + "/ws")
	fmt.Println(socket)
}
