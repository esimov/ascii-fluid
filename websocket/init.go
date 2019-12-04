package websocket

var ws = &HttpParams{
	Address: "localhost:5000",
	Prefix:  "/",
	Root:    ".",
}

func InitServer() {
	Init(ws)
}

func GetParams() *HttpParams {
	return ws
}
