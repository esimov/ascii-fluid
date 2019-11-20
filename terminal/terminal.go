package terminal

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/nsf/termbox-go"
)

type attrFunc func() (rune, termbox.Attribute, termbox.Attribute)

type Terminal struct {
	backbuf  []termbox.Cell
	bbw, bbh int
	logfile  *os.File
	fn       string
}

func New() *Terminal {
	t := new(Terminal)
	t.fn = "debug.log"
	t.logfile, _ = os.OpenFile(t.fn, os.O_CREATE|os.O_RDWR, 0755)

	return t
}

func (t *Terminal) Render() {
	defer t.logfile.Close()

	err := termbox.Init()
	if err != nil {
		panic(err)
	}
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
				t.log(t.logfile, mx, my)
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

func (t *Terminal) log(f io.Writer, vals ...interface{}) {
	fmt.Fprintf(f, "X:%d \t Y:%d\n", vals...)
}
