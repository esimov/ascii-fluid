package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

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

type hub struct {
	coords chan Detection
}

// A server application calls the Upgrade method from an HTTP request handler to initiate a connection
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var HttpServer http.Server

var detHub = &hub{
	coords: make(chan Detection),
}

const JsonFile = "facedet.json"

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
		det := &Detection{}
		err = json.Unmarshal(msg, det)
		if err != nil {
			log.Printf("error: %v", err)
		}

		detHub.coords <- *det

		if err := conn.WriteMessage(messageType, msg); err != nil {
			log.Println(err)
			return
		}
	}
}

// writeDataStream writes the detection results into the destination file
func writeDataStream(file *os.File, det Detection) error {
	data, err := json.Marshal(det)
	if err != nil {
		return err
	}
	_, err = file.WriteString(string(data) + "\n")
	if err != nil {
		return err
	}
	return nil
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

// run runs the face detection event concurrently and writes
// the detection results coordinates into the destination file
func run() {
	defer func() {
		close(detHub.coords)
	}()

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	dir := filepath.Dir(wd)
	f, err := os.Create(dir + "/" + JsonFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	for {
		select {
		case det, ok := <-detHub.coords:
			if ok {
				writeDataStream(f, det)
			}
		}
	}
}
