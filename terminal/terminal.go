package terminal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	fluid "github.com/esimov/ascii-fluid/fluid-solver"
	"github.com/esimov/ascii-fluid/websocket"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/encoding"
	runewidth "github.com/mattn/go-runewidth"
)

// Terminal is the main entry struct for the terminal based operation.
// It is also the communication bridge between the terminal and the fluid solver.
type Terminal struct {
	screen tcell.Screen
	fs     *fluid.Solver
	opts   *options
}

// options holds the fluid simulation parameters
type options struct {
	drawGrid         bool
	drawDensityField bool
	drawParticles    bool
}

type agent struct {
	x, y int
}

const (
	numOfCells         = 36 // Number of cells (not including the boundary)
	particleTimeToLive = 8
	maxNumberOfAgents  = 5
	distanceThreshold  = 80
	tickerResetTime    = 4
	minNumOfParticles  = 30
	maxNumOfParticles  = 60

	canvasWidth  = 640
	canvasHeight = 480
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

	numOfParticles = 50

	scanner *bufio.Scanner
)

var (
	termStyle  = tcell.StyleDefault.Foreground(tcell.ColorFloralWhite).Background(tcell.NewRGBColor(0, 23, 31))
	agentStyle = tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.NewRGBColor(0, 23, 31)).Dim(true)
	gridStyle  = tcell.StyleDefault.Foreground(tcell.ColorDimGray).Background(tcell.NewRGBColor(0, 23, 31)).Dim(true)
)

func init() {
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// New creates a new terminal.
func New() *Terminal {
	t := new(Terminal)
	return t
}

// Init initializes the terminal.
func (t *Terminal) Init() *Terminal {
	var err error
	t.opts = &options{
		drawGrid:         false,
		drawDensityField: true,
		drawParticles:    true,
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

// Render runs the fluid simulation in terminal, updates the screen periodically,
// handles the mouse and key events and also draws and updates the fluid particles.
func (t *Terminal) Render() {
	var (
		start      time.Time
		dx, dy     float64
		curx, cury int
	)
	wg := sync.WaitGroup{}
	mx, my := -1, -1

	quit := make(chan struct{})
	tcpConnData := make(chan string)

	// Open TCP connection.
	l, err := net.Listen("tcp", "localhost:6000")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	go func() {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		for {
			// Handle the connection in a new goroutine.
			// Multiple connections may be served concurrently.
			go func(c net.Conn) {
				reader := bufio.NewReader(c)
				for {
					data, err := reader.ReadString('\n')
					if err != nil {
						fmt.Fprintf(os.Stderr, "%v\n", err)
						return
					}
					tcpConnData <- string(data)
				}
				// Shut down the connection.
				c.Close()
			}(conn)

			ev := t.screen.PollEvent()

			switch ev := ev.(type) {
			case *tcell.EventResize:
				termWidth, termHeight = t.screen.Size()
				t.screen.Sync()
			case *tcell.EventKey:
				if ev.Key() == tcell.KeyEscape {
					// We received an interrupt signal, shut down.
					if err := websocket.HttpServer.Shutdown(context.Background()); err != nil {
						// Error from closing listeners, or context timeout:
						log.Printf("HTTP server Shutdown: %v", err)
					}
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
					if numOfParticles < maxNumOfParticles {
						numOfParticles++
					}
				case tcell.ButtonNone:
					isMouseDown = false
					isTabDown = false
				}
			}
		}
	}()

	// Sends to the channel on every second multiplied with `tickerResetTime`.
	tick := time.NewTicker(time.Second * tickerResetTime).C

loop:
	for {
		select {
		case <-quit:
			break loop
		case data := <-tcpConnData:
			det := &websocket.Detection{}
			if err := json.Unmarshal([]byte(data), det); err == nil {
				dt := time.Since(start).Seconds()
				if dt > tickerResetTime {
					curx, cury = det.X, det.Y
				}
				dx, dy = math.Abs(float64(det.X-curx)), math.Abs(float64(det.Y-cury))
				if int(dx) > distanceThreshold || int(dy) > distanceThreshold {
					isMouseDown = true
				}
				posX := int((float64(termWidth) / float64(canvasWidth)) * float64(det.X))
				posY := int((float64(termHeight) / float64(canvasHeight)) * float64(det.Y))

				t.onMouseMove(posX, posY)
			}
		case <-time.After(time.Millisecond * 10):
		case <-tick:
			start = time.Now()
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
		for i := 0; i < numOfParticles; i++ {
			p := fluid.NewParticle(
				float64(mouseX)+random(rnd, -10, 10),
				float64(mouseY)+random(rnd, -10, 10),
			)
			p.SetVy(du)
			p.SetVy(dv)

			particles = append(particles, p)
		}
	}

	// draw the fluid agents in case the tab key is pressed.
	if isTabDown {
		if i, ok := t.isAgentActive(agents, mouseX, mouseY); ok && i != -1 {
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
		if alpha < 0.001 ||
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

				p.SetVx(t.fs.GetCell("u", x0, y0) * 5)
				p.SetVy(t.fs.GetCell("v", x0, y0) * 5)

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

	if !isMouseDown {
		time.AfterFunc(time.Second, func() {
			if numOfParticles > minNumOfParticles {
				numOfParticles--
			}
		})
	}

	lastTime = time.Now()
}

// drawGrid draws the fluid grid.
func (t *Terminal) drawGrid() {
	for i := 0; i < termWidth; i++ {
		for j := 0; j < termHeight; j++ {
			t.screen.SetContent(i, j, '.', nil, gridStyle)
		}
	}
}

// drawAgent draws an agent at {x, y} position.
func (t *Terminal) drawAgent(mx, my int) {
	t.screen.SetContent(mx, my, tcell.RuneBlock, nil, agentStyle)
}

// isAgentActive verifies if an agent at {x, y} position is visible or not.
func (t *Terminal) isAgentActive(agents []agent, x, y int) (int, bool) {
	for i, agent := range agents {
		if agent.x == x && agent.y == y {
			return i, true
		}
	}
	return -1, false
}

// random generates a random float number between min and max.
func random(rnd *rand.Rand, min, max int) float64 {
	return float64(rnd.Intn(max-min) + min)
}

// debug is a helper method for printing out various information straight in the terminal.
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
