package websocket

import (
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

type HttpParams struct {
	Address string
	Prefix  string
	Root    string
}

// Init initializes the webserver and websocket connection
func (p *HttpParams) Init() {
	var err error
	p.Root, err = filepath.Abs(p.Root)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("serving %s as %s on %s", p.Root, p.Prefix, p.Address)
	http.Handle(p.Prefix, http.StripPrefix(p.Prefix, http.FileServer(http.Dir(p.Root))))
	http.HandleFunc("/ws", p.wsEndpoint)

	mux := http.DefaultServeMux.ServeHTTP
	logger := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Print(r.RemoteAddr + " " + r.Method + " " + r.URL.String())
		mux(w, r)
	})
	err = http.ListenAndServe(p.Address, logger)
	if err != nil {
		log.Fatalln(err)
	}
}

// reader listen for new messages being sent to the websocket
func (p *HttpParams) reader(conn *websocket.Conn) {
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

// wsEndpoint defines the websocket connection endpoint
func (p *HttpParams) wsEndpoint(w http.ResponseWriter, r *http.Request) {
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
	p.reader(ws)
}
