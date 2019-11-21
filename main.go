package main

import (
	"github.com/esimov/ascii-fluid/terminal"
)

func main() {
	term := terminal.New()
	term.Init().Render()
}
