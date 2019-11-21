package terminal

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
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

const numOfCells = 128 // Number of cells (not including the boundary)

var (
	rnd         *rand.Rand
	startTime   time.Time
	isMouseDown bool
	oldMouseX   int
	oldMouseY   int
	particles   []*fluid.Particle
)

func init() {
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func New() *Terminal {
	t := new(Terminal)
	t.fn = "debug.log"
	t.logfile, _ = os.OpenFile(t.fn, os.O_CREATE|os.O_RDWR, 0755)

	return t
}

func (t *Terminal) Init() *Terminal {
	startTime = time.Now()
	isMouseDown = false
	oldMouseX = 0
	oldMouseY = 0
	particles = make([]*fluid.Particle, 0)

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

	return t
}

func (t *Terminal) Render() {
	defer t.logfile.Close()
	defer termbox.Close()

	termbox.SetInputMode(termbox.InputEsc | termbox.InputMouse)
	t.reallocBackBuffer(termbox.Size())
	t.redraw(-1, -1)

mainloop:
	for {
		mx, my := -1, -1
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Key == termbox.KeyEsc {
				break mainloop
			}
		case termbox.EventMouse:
			if ev.Key == termbox.MouseLeft {
				mx, my = ev.MouseX, ev.MouseY
				t.onMouseMove(ev)
				t.log(t.logfile, "X:%d \t Y:%d\n", mx, my)
			}
			if ev.Key == termbox.MouseRight {
				isMouseDown = true
			}
		case termbox.EventResize:
			t.reallocBackBuffer(ev.Width, ev.Height)
		}
		t.redraw(mx, my)
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

func (t *Terminal) log(f io.Writer, format string, vals ...interface{}) {
	fmt.Fprintf(f, format, vals...)
}

func (t *Terminal) onMouseMove(event termbox.Event) {
	mouseX, mouseY := event.MouseX, event.MouseY
	width, height := termbox.Size()

	// Find the cell below the mouse
	i := int(math.Abs(float64(mouseX)/float64(width))*numOfCells) + 1
	j := int(math.Abs(float64(mouseY)/float64(height))*numOfCells) + 1

	// Dont overflow grid bounds
	if i > numOfCells || i < 1 || j > numOfCells || j < 1 {
		return
	}

	// Mouse velocity
	du := float64(mouseX-oldMouseX) * 1.5
	dv := float64(mouseY-oldMouseY) * 1.5

	// Add the mouse velocity to cells above, below, to the left, and to the right as well.
	t.fs.SetCell("uOld", i, j, du)
	t.log(t.logfile, "Cell: %v\n", t.fs.GetCell("uOld", i, j))
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
		t.fs.SetCell("dOld", i, j, 50)
	}

	if isMouseDown && t.opts.drawParticles {
		for k := 0; k < 5; k++ {
			p := fluid.NewParticle(mouseX+random(rnd, -50, 50), mouseY+random(rnd, -50, 50))
			p.SetX(du)
			p.SetY(dv)

			particles = append(particles, p)
		}
	}
	// Save current mouse position for next frame
	oldMouseX = mouseX
	oldMouseY = mouseY
}

func random(rnd *rand.Rand, min, max int) int {
	return rnd.Intn(max-min) + min
}
