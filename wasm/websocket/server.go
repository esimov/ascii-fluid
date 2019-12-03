package websocket

import (
	"encoding/json"
	"fmt"
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

type detection struct {
	X int `json:"row"`
	Y int `json:"col"`
}

type HttpParams struct {
	Address string
	Prefix  string
	Root    string
}

type Hub struct {
	Message chan detection
}

var MsgHub = &Hub{
	Message: make(chan detection),
}

// Init initializes the webserver and websocket connection
func Init(p *HttpParams) {
	var err error
	p.Root, err = filepath.Abs(p.Root)
	if err != nil {
		log.Fatalln(err)
	}
	go run()

	log.Printf("serving %s as %s on %s", p.Root, p.Prefix, p.Address)
	http.Handle(p.Prefix, http.StripPrefix(p.Prefix, http.FileServer(http.Dir(p.Root))))
	http.HandleFunc("/ws", wsHandler)

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

// readSocket listen for new messages being sent to the websocket
func readSocket(conn *websocket.Conn) {
	defer func() {
		conn.Close()
	}()

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			return
		}
		det := &detection{}
		err = json.Unmarshal([]byte(msg), det)
		if err != nil {
			log.Printf("error: %v", err)
		}
		log.Printf("received: X:%d Y:%d", det.X, det.Y)

		MsgHub.Message <- *det

		if err := conn.WriteMessage(messageType, msg); err != nil {
			log.Println(err)
			return
		}
	}
}

// wsHandler defines the websocket connection endpoint
func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// Upgrade the http connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}
	go readSocket(conn)
}

func run() {
	defer func() {
		close(MsgHub.Message)
	}()

	for {
		select {
		case msg, ok := <-MsgHub.Message:
			if ok {
				fmt.Println("Result:", msg)
			}
		}
	}
}
