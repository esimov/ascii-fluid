package detector

import (
	"fmt"
	"syscall/js"
)

// Detector struct holds the main components of the fetching operation.
type Detector struct {
	respChan chan []uint8
	errChan  chan error
	done     chan struct{}

	window js.Value
}

// NewDetector initializes a new constructor function.
func NewDetector() *Detector {
	var d Detector
	d.window = js.Global()

	return &d
}

// FetchCascade retrive the cascade file through a JS http connection.
// It should return the binary data as uint8 integers or err in case of an error.
func (d *Detector) FetchCascade(url string) ([]byte, error) {
	d.respChan = make(chan []uint8)
	d.errChan = make(chan error)

	promise := js.Global().Call("fetch", url)
	success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]
		response.Call("arrayBuffer").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			go func() {
				buffer := args[0]
				uint8Array := js.Global().Get("Uint8Array").New(buffer)

				jsbuf := make([]byte, uint8Array.Get("length").Int())
				js.CopyBytesToGo(jsbuf, uint8Array)
				d.respChan <- jsbuf
			}()
			return nil
		}))
		return nil
	})

	failure := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			err := fmt.Errorf("unable to fetch the cascade file: %s", args[0].String())
			d.errChan <- err
		}()
		return nil
	})

	promise.Call("then", success, failure)

	select {
	case resp := <-d.respChan:
		return resp, nil
	case err := <-d.errChan:
		return nil, err
	}
}

// Log calls the `console.log` Javascript function
func (d *Detector) Log(args ...interface{}) {
	d.window.Get("console").Call("log", args...)
}
