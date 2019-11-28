package websocket

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/websocket"
)

// A server application calls the Upgrade method from an HTTP request handler to initiate a connection
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Init initializes the webserver and websocket connection
func Init() {
	addr := flag.String("a", ":5000", "address to serve(host:port)")
	prefix := flag.String("p", "/", "prefix path under")
	root := flag.String("r", ".", "root path to serve")
	flag.Parse()

	var err error
	*root, err = filepath.Abs(*root)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("serving %s as %s on %s", *root, *prefix, *addr)
	http.Handle(*prefix, http.StripPrefix(*prefix, http.FileServer(http.Dir(*root))))
	http.HandleFunc("/ws", wsEndpoint)

	mux := http.DefaultServeMux.ServeHTTP
	logger := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Print(r.RemoteAddr + " " + r.Method + " " + r.URL.String())
		mux(w, r)
	})
	err = http.ListenAndServe(*addr, logger)
	if err != nil {
		log.Fatalln(err)
	}
}

// wsEndpoint defines the websocket connection endpoint
func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// Upgrade the http connection to a WebSocket connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	defer ws.Close()
	reader(ws)
}

// reader listen for new messages being sent to the websocket
func reader(conn *websocket.Conn) {
	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("received: %s", msg)

		if err := conn.WriteMessage(messageType, msg); err != nil {
			log.Println(err)
			return
		}
	}
}
