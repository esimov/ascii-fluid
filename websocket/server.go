package websocket

import (
	"context"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/websocket"
)

type Detection struct {
	X int `json:"row"`
	Y int `json:"col"`
}

type HttpParams struct {
	Address string
	Prefix  string
	Root    string
}

// A server application calls the Upgrade method from an HTTP request handler to initiate a connection
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// socketChan is a channel used for sending the detection results through
var socketChan = make(chan string)

// HttpServer initializes a http server and listen for connection
var HttpServer http.Server

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
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Print(r.RemoteAddr + " " + r.Method + " " + r.URL.String())
		mux(w, r)
	})
	HttpServer = http.Server{
		Addr:    p.Address,
		Handler: handler,
	}
	err = HttpServer.ListenAndServe()
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
		socketChan <- string(msg)

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

// run runs the face detection event concurrently
func run() {
	defer func() {
		close(socketChan)
	}()

	var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := d.DialContext(ctx, "tcp", "localhost:6000")
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	for {
		select {
		case det, ok := <-socketChan:
			if ok {
				// Transfer the detection results trough a TCP connection.
				if _, err := conn.Write([]byte(det + "\n")); err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}
