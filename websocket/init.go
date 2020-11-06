package websocket

var ws = &HttpParams{
	Address: "localhost:5000",
	Prefix:  "/",
	Root:    ".",
}

func GetParams() *HttpParams {
	return ws
}
