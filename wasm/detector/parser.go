package detector

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"syscall/js"
	"time"
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

// ParseCascade loads and parse the cascade file through the
// Javascript `location.href` method supported by the `js/syscall` package.
// This method will return the cascade file encoded as a byte array.
func (d *Detector) ParseCascade(path string) ([]byte, error) {
	href := js.Global().Get("location").Get("href")
	u, err := url.Parse(href.String())
	if err != nil {
		return nil, err
	}
	u.Path = path
	u.RawQuery = fmt.Sprint(time.Now().UnixNano())

	log.Println("loading cascade file: " + u.String())
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buffer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	uint8Array := js.Global().Get("Uint8Array").New(len(buffer))
	js.CopyBytesToJS(uint8Array, buffer)

	jsbuf := make([]byte, uint8Array.Get("length").Int())
	js.CopyBytesToGo(jsbuf, uint8Array)

	return jsbuf, nil
}

// Log calls the `console.log` Javascript function
func (d *Detector) Log(args ...interface{}) {
	d.window.Get("console").Call("log", args...)
}
