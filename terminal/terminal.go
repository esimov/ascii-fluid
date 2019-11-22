package terminal

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"
	"unicode/utf8"

	fluid "github.com/esimov/ascii-fluid/fluid-solver"
	"github.com/nsf/termbox-go"
)

type attrFunc func() (rune, termbox.Attribute, termbox.Attribute)

type Terminal struct {
	backbuf  []termbox.Cell
	bbw, bbh int
	logfile  *os.File
	fn       string
	fs       *fluid.FluidSolver
	opts     *options
}

type options struct {
	drawVelocityField bool
	drawDensityField  bool
	drawParticles     bool
	grayscale         bool
}

const (
	numOfCells         = 128 // Number of cells (not including the boundary)
	particleTimeToLive = 5
)

var (
	rnd       *rand.Rand
	startTime time.Time

	isMouseDown bool
	oldMouseX   int
	oldMouseY   int

	termWidth  int
	termHeight int
	cellSize   int
	particles  []*fluid.Particle

	scanner *bufio.Scanner
)

func init() {
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func New() *Terminal {
	t := new(Terminal)
	t.fn = "debug.log"
	t.logfile, _ = os.OpenFile(t.fn, os.O_CREATE|os.O_RDWR, 0755)

	file, _ := os.OpenFile("../dets", os.O_CREATE|os.O_RDWR, 0755)
	scanner = bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	return t
}

func (t *Terminal) Init() *Terminal {
	t.opts = &options{
		drawVelocityField: false,
		drawDensityField:  true,
		drawParticles:     true,
		grayscale:         false,
	}

	err := termbox.Init()
	if err != nil {
		panic(err)
	}

	t.fs = fluid.NewSolver(termbox.Size())
	t.fs.ResetVelocity()

	startTime = time.Now()
	isMouseDown = false
	oldMouseX = 0
	oldMouseY = 0
	particles = make([]*fluid.Particle, 0)

	termWidth, termHeight = termbox.Size()
	cellSize = termWidth / numOfCells

	return t
}

func (t *Terminal) Render() {
	defer t.logfile.Close()
	defer termbox.Close()

	termbox.SetInputMode(termbox.InputEsc | termbox.InputMouse)
	t.reallocBackBuffer(termbox.Size())

	wg := sync.WaitGroup{}
	ticker := time.Tick(time.Millisecond * time.Duration(5))

	eventQueue := make(chan termbox.Event)
	go func() {
		for {
			eventQueue <- termbox.PollEvent()
		}
	}()

mainloop:
	for {
		mx, my := -1, -1
		select {
		case ev := <-eventQueue:
			switch ev.Type {
			case termbox.EventKey:
				if ev.Key == termbox.KeyEsc {
					break mainloop
				}
			case termbox.EventMouse:
				if ev.Key == termbox.MouseLeft {
					isMouseDown = true
					mx, my = ev.MouseX, ev.MouseY
					t.onMouseMove(mx, my)

					//t.redraw(mx, my)
				}
			case termbox.EventResize:
				t.reallocBackBuffer(ev.Width, ev.Height)
			}
		default:
			wg.Add(1)
			<-ticker
			go func() {
				defer wg.Done()
				t.update()
				//t.retriveDetections()
			}()
			wg.Wait()
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (t *Terminal) reallocBackBuffer(w, h int) {
	t.bbw, t.bbh = w, h
	t.backbuf = make([]termbox.Cell, w*h)
}

func (t *Terminal) redraw(mx, my int) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	if mx != -1 && my != -1 {
		r, _ := utf8.DecodeRuneInString("█")
		t.backbuf[t.bbw*my+mx] = termbox.Cell{Ch: r, Fg: termbox.ColorWhite}
	}
	copy(termbox.CellBuffer(), t.backbuf)
	attrf := func() (rune, termbox.Attribute, termbox.Attribute) {
		return '█', termbox.ColorDefault, termbox.ColorWhite
	}
	r, fg, bg := attrf()
	termbox.SetCell(mx, my, r, fg, bg)
	termbox.Flush()
}

func (t *Terminal) onMouseMove(mouseX, mouseY int) {
	// Find the cell below the mouse
	i := int(math.Abs(float64(mouseX)/float64(termWidth))*numOfCells) + 1
	j := int(math.Abs(float64(mouseY)/float64(termHeight))*numOfCells) + 1

	// Dont overflow grid bounds
	if i > numOfCells || i < 1 || j > numOfCells || j < 1 {
		return
	}

	// Mouse velocity
	du := float64(mouseX-oldMouseX) * 1.5
	dv := float64(mouseY-oldMouseY) * 1.5

	// Add the mouse velocity to cells above, below, to the left, and to the right as well.
	t.fs.SetCell("uOld", i, j, du)
	//t.log(t.logfile, "Cell: %v\n", t.fs.GetCell("uOld", i, j))
	t.fs.SetCell("vOld", i, j, dv)

	t.fs.SetCell("uOld", i+1, j, du)
	t.fs.SetCell("vOld", i+1, j, dv)

	t.fs.SetCell("uOld", i-1, j, du)
	t.fs.SetCell("vOld", i-1, j, dv)

	t.fs.SetCell("uOld", i, j+1, du)
	t.fs.SetCell("vOld", i, j+1, dv)

	t.fs.SetCell("uOld", i, j-1, du)
	t.fs.SetCell("vOld", i, j-1, dv)

	if isMouseDown {
		// If holding down the right mouse, add density to the cell below the mouse
		t.fs.SetCell("dOld", i, j, 10)
	}

	if isMouseDown && t.opts.drawParticles {
		for i := 0; i < 5; i++ {
			p := fluid.NewParticle(
				float64(mouseX)+random(rnd, -10, 10),
				float64(mouseY)+random(rnd, -10, 10),
			)
			p.SetVy(du)
			p.SetVy(dv)

			particles = append(particles, p)
		}
	}
	// Save current mouse position for next frame
	oldMouseX = mouseX
	oldMouseY = mouseY
}

func (t *Terminal) update() {
	endTime := time.Now()
	dt := endTime.Sub(startTime).Seconds()

	t.fs.VelocityStep()
	t.fs.DensityStep()

	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	if t.opts.drawVelocityField {
		// TODO implement me
	}

	for i := 0; i < len(particles); i++ {
		p := particles[i]
		p.SetAge(float64(p.GetAge()) + dt)

		alpha := float64(1 - p.GetAge()/particleTimeToLive)
		if alpha < 0.001 ||
			p.GetAge() >= particleTimeToLive ||
			p.GetX() <= 0 || p.GetX() >= termWidth ||
			p.GetY() <= 0 || p.GetY() >= termHeight {
			p.SetDeath(true)
		} else {
			x0 := int(math.Abs(float64(p.GetX())/float64(termWidth))*numOfCells) + 2
			y0 := int(math.Abs(float64(p.GetY())/float64(termHeight))*numOfCells) + 2

			p.SetVx(t.fs.GetCell("u", x0, y0) * 50)
			p.SetVy(t.fs.GetCell("v", x0, y0) * 50)

			p.SetX(float64(p.GetX() + p.GetVx()))
			p.SetY(float64(p.GetY() + p.GetVy()))

			attrf := func() (rune, termbox.Attribute, termbox.Attribute) {
				return '·', termbox.ColorDefault, termbox.ColorDefault
			}
			r, fg, bg := attrf()
			termbox.SetCell(p.GetX(), p.GetY(), r, fg, bg)
		}

		if p.GetDeath() {
			// Remove dead particles, and update the length manually
			particles = append(particles[:i], particles[i+1:]...)
		}
	}
	startTime = time.Now()

	// // Render fluid
	// for i := 1; i <= numOfCells; i++ {
	// 	// the x position of the current cell
	// 	//dx := (float64(i) - 0.5) * float64(cellSize)

	// 	for j := 1; j <= numOfCells; j++ {
	// 		// the y position of the current cell
	// 		//dy := (float64(j) - 0.5) * float64(cellSize)

	// 		//density := t.fs.GetCell("d", i, j)
	// 		if t.opts.drawDensityField {
	// 			for l := 0; l < cellSize; l++ {
	// 				for m := 0; m < cellSize; m++ {
	// 					mx := (i-1)*cellSize + l
	// 					my := (j-1)*cellSize + m
	// 					//pxIdx := pxX + pxY*terminalSize*4

	// 					t.log(t.logfile, "x:%v\t y:%v\n", mx, my)

	// 					attrf := func() (rune, termbox.Attribute, termbox.Attribute) {
	// 						return '█', termbox.ColorDefault, termbox.ColorWhite
	// 					}
	// 					r, fg, bg := attrf()
	// 					termbox.SetCell(mx, my, r, fg, bg)
	// 				}
	// 			}
	// 		}

	// 	}
	// }
	termbox.Flush()
}

func (t *Terminal) retriveDetections() {
	for scanner.Scan() { // internally, it advances token based on sperator
		fmt.Println(scanner.Text())  // token in unicode-char
		fmt.Println(scanner.Bytes()) // token in bytes

	}
}

func (t *Terminal) log(f io.Writer, format string, vals ...interface{}) {
	fmt.Fprintf(f, format, vals...)
}

func random(rnd *rand.Rand, min, max int) float64 {
	return float64(rnd.Intn(max-min) + min)
}
