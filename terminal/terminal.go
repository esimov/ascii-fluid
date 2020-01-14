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

	fluid "github.com/esimov/ascii-fluid/fluid-solver"
	"github.com/esimov/ascii-fluid/websocket"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/encoding"
	runewidth "github.com/mattn/go-runewidth"
)

type Terminal struct {
	screen  tcell.Screen
	logfile *os.File
	fn      string
	fs      *fluid.FluidSolver
	opts    *options
}

type options struct {
	drawVelocityField bool
	drawDensityField  bool
	drawParticles     bool
	grayscale         bool
}

const (
	numOfCells         = 64 // Number of cells (not including the boundary)
	particleTimeToLive = 5
)

var (
	rnd      *rand.Rand
	lastTime time.Time

	isMouseDown bool
	oldMouseX   int
	oldMouseY   int

	termWidth  int
	termHeight int
	cellSize   int
	particles  []*fluid.Particle

	scanner *bufio.Scanner
)

var style = tcell.StyleDefault.
	Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)

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
	var err error
	t.opts = &options{
		drawVelocityField: false,
		drawDensityField:  true,
		drawParticles:     true,
		grayscale:         false,
	}

	lastTime = time.Now()
	isMouseDown = false
	oldMouseX = 0
	oldMouseY = 0
	particles = make([]*fluid.Particle, 0)

	t.screen, err = tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	encoding.Register()

	if e := t.screen.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	defStyle := tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorWhite)
	t.screen.SetStyle(defStyle)
	t.screen.EnableMouse()
	t.screen.Clear()

	termWidth, termHeight = t.screen.Size()
	cellSize = termWidth / numOfCells

	t.fs = fluid.NewSolver(numOfCells)
	t.fs.ResetVelocity()

	return t
}

func (t *Terminal) Render() {
	wg := sync.WaitGroup{}
	mx, my := -1, -1
	//posfmt := "Mouse: %d, %d  "

	defer t.logfile.Close()

	for {
		//print(t.screen, 2, 3, style, fmt.Sprintf(posfmt, mx, my))
		ev := t.screen.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventResize:
			t.screen.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape {
				t.screen.Fini()
				os.Exit(0)
			}
		case *tcell.EventMouse:
			mx, my = ev.Position()
			t.onMouseMove(mx, my)

			switch ev.Buttons() {
			case tcell.Button1:
				isMouseDown = true
			case tcell.ButtonNone:
				isMouseDown = false
			}
		default:
			go t.getDetectionResults()
		}

		t.screen.Show()

		wg.Add(1)
		go func() {
			defer wg.Done()
			// Clear the screen
			t.screen.Fill(' ', style)
			t.update()
		}()
		wg.Wait()
	}
}

func (t *Terminal) onMouseMove(mouseX, mouseY int) {
	// Find the cell below the mouse
	i := int(math.Abs(float64(mouseX)/float64(termWidth))*numOfCells) + 1
	j := int(math.Abs(float64(mouseY)/float64(termHeight))*numOfCells) + 1

	// Don't overflow grid bounds
	if i > numOfCells || i < 1 || j > numOfCells || j < 1 {
		return
	}

	// Mouse velocity
	du := float64(mouseX-oldMouseX) * 1.5
	dv := float64(mouseY-oldMouseY) * 1.5

	print(t.screen, 2, 3, style, fmt.Sprintf("Velocity: %v, %v", du, dv))

	// Add the mouse velocity to cells above, below, to the left, and to the right as well.
	t.fs.SetCell("uOld", i, j, du)
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
		// If holding down the mouse, add density to the cell below the mouse
		t.fs.SetCell("dOld", i, j, 50)
	}

	if isMouseDown && t.opts.drawParticles {
		for i := 0; i < 5; i++ {
			p := fluid.NewParticle(
				float64(mouseX)+random(rnd, -20, 20),
				float64(mouseY)+random(rnd, -20, 20),
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
	dt := time.Now().Sub(lastTime).Seconds()
	//print(t.screen, 2, 5, style, fmt.Sprintf("UPDATE %v", dt))

	t.fs.VelocityStep()
	t.fs.DensityStep()

	if t.opts.drawVelocityField {
		// TODO implement me
	}

	for i := 0; i < len(particles); i++ {
		p := particles[i]
		p.SetAge(float64(p.GetAge()) + dt)

		alpha := float64(1 - p.GetAge()/particleTimeToLive)
		if alpha < 0.001 ||
			p.GetAge() >= particleTimeToLive ||
			p.GetX() <= 0.0 || p.GetX() >= float64(termWidth) ||
			p.GetY() <= 0.0 || p.GetY() >= float64(termHeight) {
			p.SetDeath(true)
		} else {
			x0 := int(math.Abs(float64(p.GetX())/float64(termWidth))*numOfCells) + 2
			y0 := int(math.Abs(float64(p.GetY())/float64(termHeight))*numOfCells) + 2

			//print(t.screen, 2, 4, style, fmt.Sprintf("Particle: %v %v", x0, y0))

			p.SetVx(t.fs.GetCell("u", x0, y0) * 50)
			p.SetVy(t.fs.GetCell("v", x0, y0) * 50)

			//print(t.screen, 2, 6, style, fmt.Sprintf("U cell: %v", t.fs.GetCell("u", x0, y0)*50))
			print(t.screen, 2, 7, style, fmt.Sprintf("U cell: %v", p.GetVx()))

			p.SetX(float64(p.GetX() + p.GetVx()))
			p.SetY(float64(p.GetY() + p.GetVy()))
			t.screen.SetContent(int(p.GetX()), int(p.GetY()), '·', nil, tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack))

			//print(t.screen, 2, 5, style, fmt.Sprintf("Velocity: %v %v", p.GetX(), p.GetY()))
		}
		//print(t.screen, 2, 4, style, fmt.Sprintf("Particle: %v", alpha))

		if p.GetDeath() {
			// Remove dead particles, and update the length manually
			particles = append(particles[:i], particles[i+1:]...)
		}
	}
	lastTime = time.Now()

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
}

func (t *Terminal) getDetectionResults() {
	defer func() {
		close(websocket.MsgHub.Message)
	}()

	for {
		fmt.Println(websocket.TestVar)
		fmt.Println(<-websocket.MsgHub.Message)
		select {
		case msg, ok := <-websocket.MsgHub.Message:
			if !ok {
				fmt.Println("closed")
			}
			fmt.Println(msg)
		}
	}
}

func (t *Terminal) log(f io.Writer, format string, vals ...interface{}) {
	fmt.Fprintf(f, format, vals...)
}

func random(rnd *rand.Rand, min, max int) float64 {
	return float64(rnd.Intn(max-min) + min)
}

func print(s tcell.Screen, x, y int, style tcell.Style, str string) {
	for _, c := range str {
		var comb []rune
		w := runewidth.RuneWidth(c)
		if w == 0 {
			comb = []rune{c}
			c = ' '
			w = 1
		}
		s.SetContent(x, y, c, comb, style)
		x += w
	}
}
