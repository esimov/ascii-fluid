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
	screen tcell.Screen
	fn     string
	fs     *fluid.FluidSolver
	opts   *options
}

type options struct {
	drawGrid         bool
	drawDensityField bool
	drawParticles    bool
	grayscale        bool
}

type agent struct {
	x, y int
}

const (
	numOfCells         = 32 // Number of cells (not including the boundary)
	particleTimeToLive = 3
	maxNumberOfAgents  = 10
)

var (
	rnd      *rand.Rand
	lastTime time.Time

	isMouseDown bool
	isTabDown   bool
	oldMouseX   int
	oldMouseY   int

	termWidth  int
	termHeight int
	cellSize   int
	particles  []*fluid.Particle
	agents     []agent

	scanner *bufio.Scanner
)

var (
	termStyle  = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	agentStyle = tcell.StyleDefault.Foreground(tcell.ColorGreenYellow).Background(tcell.ColorBlack)
	gridStyle  = tcell.StyleDefault.Foreground(tcell.ColorDimGray).Background(tcell.ColorBlack)
)

func init() {
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func New() *Terminal {
	t := new(Terminal)
	return t
}

func (t *Terminal) Init() *Terminal {
	var err error
	t.opts = &options{
		drawGrid:         true,
		drawDensityField: true,
		drawParticles:    true,
		grayscale:        false,
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
	t.screen.SetStyle(termStyle)
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

	quit := make(chan struct{})
	go func() {
		for {
			ev := t.screen.PollEvent()

			switch ev := ev.(type) {
			case *tcell.EventResize:
				t.screen.Sync()
			case *tcell.EventKey:
				if ev.Key() == tcell.KeyEscape {
					close(quit)
					return
				}
				if ev.Key() == tcell.KeyCtrlD {
					t.opts.drawGrid = !t.opts.drawGrid
				}
				if ev.Key() == tcell.KeyTAB && isMouseDown {
					isTabDown = true
				}
			case *tcell.EventMouse:
				mx, my = ev.Position()
				t.onMouseMove(mx, my)

				switch ev.Buttons() {
				case tcell.Button1:
					isMouseDown = true
				case tcell.ButtonNone:
					isMouseDown = false
					isTabDown = false
				}
			default:
				go t.getDetectionResults()
			}
		}
	}()

loop:
	for {
		select {
		case <-quit:
			break loop
		case <-time.After(time.Millisecond * 5):
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Clear the screen
			t.screen.Fill(' ', termStyle)
			t.update()
		}()
		wg.Wait()
		t.screen.Show()
	}
	t.screen.Fini()
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
		// Add density to the cell below the mouse
		t.fs.SetCell("dOld", i, j, 50)
	}

	if isMouseDown && t.opts.drawParticles {
		for i := 0; i < 20; i++ {
			p := fluid.NewParticle(
				float64(mouseX)+random(rnd, -10, 10),
				float64(mouseY)+random(rnd, -10, 10),
			)
			p.SetVy(du)
			p.SetVy(dv)

			particles = append(particles, p)
		}
	}

	if isTabDown {
		if i, ok := isAgentActive(agents, mouseX, mouseY); ok && i != -1 {
			// remove agent
			agents = append(agents[:i], agents[i+1:]...)
		} else {
			// add agent
			if len(agents) < maxNumberOfAgents {
				agents = append(agents, agent{x: mouseX, y: mouseY})
			}
		}
	}

	// Save current mouse position for next frame
	oldMouseX = mouseX
	oldMouseY = mouseY
}

func (t *Terminal) update() {
	dt := time.Now().Sub(lastTime).Seconds()

	t.fs.VelocityStep()
	t.fs.DensityStep()

	if t.opts.drawGrid {
		t.drawGrid()
	}

	for i := 0; i < len(particles); i++ {
		p := particles[i]
		p.SetAge(float64(p.GetAge()) + dt)

		alpha := float64(1 - p.GetAge()/particleTimeToLive)
		//debug(t.screen, 2, 1, termStyle, fmt.Sprintf("Alpha: %v", alpha))
		if alpha < 0.01 ||
			p.GetAge() >= particleTimeToLive ||
			p.GetX() <= 0.0 || p.GetX() >= float64(termWidth) ||
			p.GetY() <= 0.0 || p.GetY() >= float64(termHeight) {
			p.SetDeath(true)
		} else {
			x0 := int(math.Abs(float64(p.GetX())/float64(termWidth))*numOfCells) + 2
			y0 := int(math.Abs(float64(p.GetY())/float64(termHeight))*numOfCells) + 2

			p.SetVx(t.fs.GetCell("u", x0, y0) * 50)
			p.SetVy(t.fs.GetCell("v", x0, y0) * 50)

			p.SetX(float64(p.GetX() + p.GetVx()))
			p.SetY(float64(p.GetY() + p.GetVy()))

			// Apply a velocity factor to the existing agents
			for i := 0; i < len(agents); i++ {
				x0 := int(math.Abs(float64(agents[i].x)/float64(termWidth))*numOfCells) + 2
				y0 := int(math.Abs(float64(agents[i].y)/float64(termHeight))*numOfCells) + 2

				p.SetVx(t.fs.GetCell("u", x0, y0) * 10)
				p.SetVy(t.fs.GetCell("v", x0, y0) * 10)

				p.SetX(float64(p.GetX() + p.GetVx()))
				p.SetY(float64(p.GetY() + p.GetVy()))

			}
			t.screen.SetContent(int(p.GetX()), int(p.GetY()), '▄', nil, termStyle)
		}

		if p.GetDeath() {
			// Remove dead particles, and update the length manually
			particles = append(particles[:i], particles[i+1:]...)
		}
	}

	for i := 0; i < len(agents); i++ {
		t.drawAgent(agents[i].x, agents[i].y)
	}

	lastTime = time.Now()
}

func (t *Terminal) drawGrid() {
	for i := 0; i < termWidth; i++ {
		for j := 0; j < termHeight; j++ {
			t.screen.SetContent(i, j, '.', nil, gridStyle)
		}
	}
}

func (t *Terminal) drawAgent(mx, my int) {
	t.screen.SetContent(mx, my, tcell.RuneBlock, nil, agentStyle)
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

func isAgentActive(agents []agent, x, y int) (int, bool) {
	for i, agent := range agents {
		if agent.x == x && agent.y == y {
			return i, true
		}
	}
	return -1, false
}

func debug(s tcell.Screen, x, y int, style tcell.Style, str string) {
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
