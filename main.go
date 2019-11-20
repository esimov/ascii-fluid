package main

import (
	fluid "github.com/esimov/ascii-fluid/fluid-solver"
	"github.com/esimov/ascii-fluid/terminal"
)

const numOfCells = 128 // Number of cells (not including the boundary)

func main() {
	term := terminal.New()
	term.Render()

	fs := fluid.NewSolver(numOfCells)
	fs.ResetVelocity()
}
