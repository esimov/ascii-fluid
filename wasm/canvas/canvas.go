package canvas

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"syscall/js"

	"github.com/esimov/ascii-fluid/wasm/detector"
)

// Canvas struct holds the Javascript objects needed for the Canvas creation
type Canvas struct {
	dets   chan [][]int
	done   chan struct{}
	succCh chan struct{}
	errCh  chan error

	// DOM elements
	window     js.Value
	doc        js.Value
	body       js.Value
	windowSize struct{ width, height int }

	// Canvas properties
	canvas   js.Value
	ctx      js.Value
	reqID    js.Value
	renderer js.Func

	// Webcam properties
	navigator js.Value
	video     js.Value

	showPupil  bool
	drawCircle bool

	// Websocket
	ws js.Value
}

const (
	canvasWidth  = 640
	canvasHeight = 480
)

var det *detector.Detector

// NewCanvas creates and initializes a new canvas element
func NewCanvas() *Canvas {
	var c Canvas
	c.window = js.Global()
	c.doc = c.window.Get("document")
	c.body = c.doc.Get("body")

	c.windowSize.width = canvasWidth
	c.windowSize.height = canvasHeight

	c.canvas = c.doc.Call("createElement", "canvas")
	c.canvas.Set("width", c.windowSize.width)
	c.canvas.Set("height", c.windowSize.height)
	c.canvas.Set("id", "canvas")
	c.body.Call("appendChild", c.canvas)

	c.ctx = c.canvas.Call("getContext", "2d")
	c.showPupil = false
	c.drawCircle = false

	det = detector.NewDetector()

	c.InitWebSocket()
	return &c
}

// Render calls the `requestAnimationFrame` Javascript function in asynchronous mode.
func (c *Canvas) Render() error {
	width, height := c.windowSize.width, c.windowSize.height
	var data = make([]byte, width*height*4)
	c.done = make(chan struct{})

	err := det.UnpackCascades()
	if err != nil {
		return err
	}

	c.renderer = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			width, height := c.windowSize.width, c.windowSize.height
			c.reqID = c.window.Call("requestAnimationFrame", c.renderer)
			// Draw the webcam frame to the canvas element
			c.ctx.Call("drawImage", c.video, 0, 0)
			rgba := c.ctx.Call("getImageData", 0, 0, width, height).Get("data")

			uint8Arr := js.Global().Get("Uint8Array").New(rgba)
			js.CopyBytesToGo(data, uint8Arr)
			pixels := c.rgbaToGrayscale(data)

			// Restore the data slice to its default values to avoid unnecessary memory allocation.
			// Otherwise, the GC won't clean up the memory address allocated by this slice
			// and the memory will keep up increasing by each iteration.
			data = make([]byte, len(data))

			res := det.DetectFaces(pixels, height, width)
			if len(res) > 0 {
				fmt.Println(res)
				c.drawDetection(res)
				c.send(string(c.stringify(res[0])))
			}
		}()
		return nil
	})
	// Release renderer to free up resources.
	defer c.renderer.Release()

	c.window.Call("requestAnimationFrame", c.renderer)
	c.detectKeyPress()
	<-c.done

	return nil
}

// StartWebcam reads the webcam data and feeds it into the canvas element.
// It returns an empty struct in case of success and error in case of failure.
func (c *Canvas) StartWebcam() (*Canvas, error) {
	var err error
	c.succCh = make(chan struct{})
	c.errCh = make(chan error)

	c.video = c.doc.Call("createElement", "video")

	// If we don't do this, the stream will not be played.
	c.video.Set("autoplay", 1)
	c.video.Set("playsinline", 1) // important for iPhones

	// The video should fill out all of the canvas
	c.video.Set("width", 0)
	c.video.Set("height", 0)

	c.body.Call("appendChild", c.video)

	success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			c.video.Set("srcObject", args[0])
			c.video.Call("play")
			c.succCh <- struct{}{}
		}()
		return nil
	})

	failure := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			err = fmt.Errorf("failed initialising the camera: %s", args[0].String())
			c.errCh <- err
		}()
		return nil
	})

	opts := js.Global().Get("Object").New()

	videoSize := js.Global().Get("Object").New()
	videoSize.Set("width", c.windowSize.width)
	videoSize.Set("height", c.windowSize.height)
	videoSize.Set("aspectRatio", 1.777777778)

	opts.Set("video", videoSize)
	opts.Set("audio", false)

	promise := c.window.Get("navigator").Get("mediaDevices").Call("getUserMedia", opts)
	promise.Call("then", success, failure)

	select {
	case <-c.succCh:
		return c, nil
	case err := <-c.errCh:
		return nil, err
	}
}

// Stop stops the rendering.
func (c *Canvas) Stop() {
	c.window.Call("cancelAnimationFrame", c.reqID)
	c.done <- struct{}{}
	close(c.done)
}

func (c *Canvas) stringify(res []int) []byte {
	x, y := res[1], res[0]
	result := c.newDetection(x, y)

	json, err := json.Marshal(result)
	if err != nil {
		log.Println(err)
	}
	return json
}

// rgbaToGrayscale converts the rgb pixel values to grayscale
func (c *Canvas) rgbaToGrayscale(data []uint8) []uint8 {
	rows, cols := c.windowSize.width, c.windowSize.height
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			// gray = 0.2*red + 0.7*green + 0.1*blue
			data[r*cols+c] = uint8(math.Round(
				0.2126*float64(data[r*4*cols+4*c+0]) +
					0.7152*float64(data[r*4*cols+4*c+1]) +
					0.0722*float64(data[r*4*cols+4*c+2])))
		}
	}
	return data
}

// drawDetection draws the detected faces and eyes.
func (c *Canvas) drawDetection(dets [][]int) {
	for i := 0; i < len(dets); i++ {
		if dets[i][3] > 50 {
			c.ctx.Call("beginPath")
			c.ctx.Set("lineWidth", 3)
			c.ctx.Set("strokeStyle", "red")

			row, col, scale := dets[i][1], dets[i][0], dets[i][2]
			if c.drawCircle {
				c.ctx.Call("moveTo", row+int(scale/2), col)
				c.ctx.Call("arc", row, col, scale/2, 0, 2*math.Pi, true)
			} else {
				c.ctx.Call("rect", row-scale/2, col-scale/2, scale, scale)
			}
			c.ctx.Call("stroke")

			if c.showPupil {
				leftPupil := det.DetectLeftPupil(dets[i])
				if leftPupil != nil {
					col, row, scale := leftPupil.Col, leftPupil.Row, leftPupil.Scale/8
					c.ctx.Call("moveTo", col+int(scale), row)
					c.ctx.Call("arc", col, row, scale, 0, 2*math.Pi, true)
				}

				rightPupil := det.DetectRightPupil(dets[i])
				if rightPupil != nil {
					col, row, scale := rightPupil.Col, rightPupil.Row, leftPupil.Scale/8
					c.ctx.Call("moveTo", col+int(scale), row)
					c.ctx.Call("arc", col, row, scale, 0, 2*math.Pi, true)
				}
				c.ctx.Call("stroke")
			}
		}
	}
}

// detectKeyPress listen for the keypress event and retrieves the key code.
func (c *Canvas) detectKeyPress() {
	keyEventHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		keyCode := args[0].Get("key")
		switch {
		case keyCode.String() == "s":
			c.showPupil = !c.showPupil
		case keyCode.String() == "c":
			c.drawCircle = !c.drawCircle
		default:
			c.drawCircle = false
		}
		return nil
	})
	c.doc.Call("addEventListener", "keypress", keyEventHandler)
}

// Log calls the `console.log` Javascript function
func (c *Canvas) Log(args ...interface{}) {
	c.window.Get("console").Call("log", args...)
}

// Alert calls the `alert` Javascript function
func (c *Canvas) Alert(args ...interface{}) {
	alert := c.window.Get("alert")
	alert.Invoke(args...)
}
