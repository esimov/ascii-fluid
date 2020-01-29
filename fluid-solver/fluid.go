// The fluid solver implementation is largely based on Jos Stam's paper "Real-Time Fluid Dynamics for Games".
// @link http://www.dgp.toronto.edu/people/stam/reality/Research/pdf/GDC03.pdf

package fluid

import (
	"math"
)

type cell []float64

type solver struct {
	nx          int
	ny          int
	dt          float64
	diffusion   float64
	viscosity   float64
	iterations  int
	doVorticity bool
	doBuoyancy  bool
	numOfCells  int

	u cell
	v cell
	d cell

	uOld cell
	vOld cell
	dOld cell

	curlData cell
}

// BoundaryType is a type alias for int
type BoundaryType int

const (
	BoundaryNone BoundaryType = iota
	BoundaryLeftRight
	BoundaryTopBottom
)

// Solver is a global alias to solver for using outside of this package.
type Solver solver

// NewSolver defines the fluid solver general parameters, where {n} is the
// number of fluid cells for the simulation grid in each dimension (NxN)
func NewSolver(n int) *Solver {
	fs := &Solver{
		nx:          n,
		ny:          n,
		dt:          0.2,
		diffusion:   0.0001,
		viscosity:   0.0,
		iterations:  10,
		doVorticity: true,
		doBuoyancy:  true,
	}
	fs.numOfCells = (n + 2) * (n + 2)
	fs.u = make(cell, fs.numOfCells)
	fs.v = make(cell, fs.numOfCells)
	fs.d = make(cell, fs.numOfCells)

	fs.uOld = make(cell, fs.numOfCells)
	fs.vOld = make(cell, fs.numOfCells)
	fs.dOld = make(cell, fs.numOfCells)

	fs.curlData = make(cell, fs.numOfCells)

	return fs
}

// SetCell sets the cell value of different types.
func (fs *Solver) SetCell(cellType interface{}, x, y int, val float64) {
	switch cellType {
	case "u":
		fs.u[fs.idx(x, y)] = val
	case "v":
		fs.v[fs.idx(x, y)] = val
	case "d":
		fs.d[fs.idx(x, y)] = val
	case "uOld":
		fs.uOld[fs.idx(x, y)] = val
	case "vOld":
		fs.vOld[fs.idx(x, y)] = val
	case "dOld":
		fs.dOld[fs.idx(x, y)] = val
	}
}

// GetCell gets the cell value of different types.
func (fs *Solver) GetCell(cellType interface{}, x, y int) (result float64) {
	switch cellType {
	case "u":
		result = fs.u[fs.idx(x, y)]
	case "v":
		result = fs.v[fs.idx(x, y)]
	case "d":
		result = fs.d[fs.idx(x, y)]
	case "uOld":
		result = fs.uOld[fs.idx(x, y)]
	case "vOld":
		result = fs.vOld[fs.idx(x, y)]
	case "dOld":
		result = fs.dOld[fs.idx(x, y)]
	}
	return
}

// DensityStep calculates the density step.
func (fs *Solver) DensityStep() {
	fs.addSource(fs.d, fs.dOld)

	fs.swapD()
	fs.diffuse(BoundaryNone, fs.d, fs.dOld, fs.diffusion)

	fs.swapD()
	fs.advect(BoundaryNone, fs.d, fs.dOld, fs.u, fs.v)

	// reset for the next step
	for i := 0; i < fs.numOfCells; i++ {
		fs.dOld[i] = 0
	}
}

// VelocityStep calculates the velocity step.
func (fs *Solver) VelocityStep() {
	fs.addSource(fs.u, fs.uOld)
	fs.addSource(fs.v, fs.vOld)

	if fs.doVorticity {
		fs.calcVorticityConfinement(fs.uOld, fs.vOld)
		fs.addSource(fs.u, fs.uOld)
		fs.addSource(fs.v, fs.vOld)
	}

	if fs.doBuoyancy {
		fs.buoyancy(fs.vOld)
		fs.addSource(fs.v, fs.vOld)
	}

	fs.swapU()
	fs.diffuse(BoundaryLeftRight, fs.u, fs.uOld, fs.viscosity)

	fs.swapV()
	fs.diffuse(BoundaryTopBottom, fs.v, fs.vOld, fs.viscosity)

	fs.project(fs.u, fs.v, fs.uOld, fs.vOld)
	fs.swapU()
	fs.swapV()

	fs.advect(BoundaryLeftRight, fs.u, fs.uOld, fs.uOld, fs.vOld)
	fs.advect(BoundaryTopBottom, fs.v, fs.vOld, fs.uOld, fs.vOld)

	fs.project(fs.u, fs.v, fs.uOld, fs.vOld)

	// reset for the next step
	for i := 0; i < fs.numOfCells; i++ {
		fs.uOld[i] = 0
		fs.vOld[i] = 0
	}
}

// ResetDensity resets the density cells.
func (fs *Solver) ResetDensity() {
	for i := 0; i < fs.numOfCells; i++ {
		fs.d[i] = 0
	}
}

// ResetVelocity resets the velocity cells.
func (fs *Solver) ResetVelocity() {
	for i := 0; i < fs.numOfCells; i++ {
		// Set a small value so we can render the velocity field
		fs.v[i] = 0.001
		fs.u[i] = 0.001
	}
}

// swapU swaps velocity x reference.
func (fs *Solver) swapU() {
	tmp := fs.u
	fs.u = fs.uOld
	fs.uOld = tmp
}

// swapU swaps velocity y reference.
func (fs *Solver) swapV() {
	tmp := fs.v
	fs.v = fs.vOld
	fs.vOld = tmp
}

// swapU swaps density reference.
func (fs *Solver) swapD() {
	tmp := fs.d
	fs.d = fs.dOld
	fs.dOld = tmp
}

// addSource integrates the density sources.
func (fs *Solver) addSource(x, s cell) {
	for i := 0; i < fs.numOfCells; i++ {
		x[i] += s[i] * fs.dt
	}
}

// curls calculates the curl at cell (i, j).
func (fs *Solver) curl(i, j int) float64 {
	duDy := fs.u[fs.idx(i, j+1)] - fs.u[fs.idx(i, j-1)]*0.5
	dvDx := fs.v[fs.idx(i+1, j)] - fs.v[fs.idx(i-1, j)]*0.5

	return duDy - dvDx
}

// calcVorticityConfinement calculates the vorticity confinement force for each cell.
func (fs *Solver) calcVorticityConfinement(x, y cell) {
	var (
		i, j            int
		dx, dy, norm, v float64
	)

	for i = 1; i <= fs.nx; i++ {
		for j = 1; j <= fs.ny; j++ {
			// Calculate the magnitude of curl(i, j) for each cell
			fs.curlData[fs.idx(i, j)] = math.Abs(fs.curl(i, j))

			dx = fs.curlData[fs.idx(i+1, j)] - fs.curlData[fs.idx(i-1, j)]*0.5
			dy = fs.curlData[fs.idx(i, j+1)] - fs.curlData[fs.idx(i, j-1)]*0.5

			norm = math.Sqrt((dx * dx) + (dy * dy))
			if norm == 0 {
				// Avoid devide by zero
				norm = 1
			}
			dx /= norm
			dy /= norm

			v = fs.curl(i, j)

			x[fs.idx(i, j)] = dy * v * -1
			y[fs.idx(i, j)] = dx * v
		}
	}
}

// buoyancy calculates the buoyancy force for the grid.
func (fs *Solver) buoyancy(buoy cell) cell {
	var (
		i, j int
		tAmb float64
		a    = 0.000625
		b    = 0.015
	)

	// Sum all temperatures (faster)
	for i := 0; i < len(fs.d); i++ {
		tAmb += fs.d[i]
	}

	// Calculate the average temperature of the grid
	tAmb /= float64(fs.nx * fs.ny)

	// For each cell compute the bouyancy force
	for i = 1; i <= fs.nx; i++ {
		for j = 1; j <= fs.ny; j++ {
			buoy[fs.idx(i, j)] = a*fs.d[fs.idx(i, j)] + -b*(fs.d[fs.idx(i, j)]-tAmb)
		}
	}
	return buoy
}

// diffuse diffuses the density between neighbouring cells.
func (fs *Solver) diffuse(bound BoundaryType, x, x0 cell, diffusion float64) {
	a := fs.dt * diffusion * float64(fs.nx*fs.ny)
	fs.linearSolve(bound, x, x0, a, 1.0+4.0*a)
}

func (fs *Solver) linearSolve(bound BoundaryType, x, x0 cell, a, c float64) {
	invC := 1.0 / c

	for k := 0; k < fs.iterations; k++ {
		for i := 1; i <= fs.nx; i++ {
			for j := 1; j <= fs.ny; j++ {
				x[fs.idx(i, j)] = (x0[fs.idx(i, j)] + a*(x[fs.idx(i-1, j)]+x[fs.idx(i+1, j)]+x[fs.idx(i, j-1)]+x[fs.idx(i, j+1)])) * invC
			}
		}
		fs.setBoundary(bound, x)
	}
}

// project solves the Poisson Equation.
func (fs *Solver) project(u, v, p, div cell) {
	// Calculate the gradient field
	h := 1.0 / float64(fs.ny)
	for i := 1; i <= fs.nx; i++ {
		for j := 1; j <= fs.ny; j++ {
			div[fs.idx(i, j)] = -0.5 * h * (u[fs.idx(i+1, j)] - u[fs.idx(i-1, j)] + v[fs.idx(i, j+1)] - v[fs.idx(i, j-1)])
			p[fs.idx(i, j)] = 0
		}
	}
	fs.setBoundary(BoundaryNone, div)
	fs.setBoundary(BoundaryNone, p)

	// Solve the Poisson equation
	fs.linearSolve(BoundaryNone, p, div, 1, 4)

	// Substract the gradient field from the velocity field to get the mass conserving velocity field.
	for i := 1; i <= fs.nx; i++ {
		for j := 1; j <= fs.ny; j++ {
			u[fs.idx(i, j)] = 0.5 * (p[fs.idx(i+1, j)] - p[fs.idx(i-1, j)]) / h
			v[fs.idx(i, j)] = 0.5 * (p[fs.idx(i, j+1)] - p[fs.idx(i, j-1)]) / h
		}
	}
	fs.setBoundary(BoundaryLeftRight, u)
	fs.setBoundary(BoundaryTopBottom, v)
}

// advect moves the density through the static velocity field.
func (fs *Solver) advect(bound BoundaryType, d, d0, u, v cell) {
	var (
		i, j, i0, j0, i1, j1 int
		x, y, s0, t0, s1, t1 float64
		dt0, dt1             float64
	)
	dt0 = fs.dt * float64(fs.nx)
	dt1 = fs.dt * float64(fs.ny)

	for i = 1; i <= fs.nx; i++ {
		for j = 1; j <= fs.ny; j++ {
			x = float64(i) - dt0*u[fs.idx(i, j)]
			y = float64(j) - dt1*v[fs.idx(i, j)]
			if x < 0.5 {
				x = 0.5
			}
			if x > float64(fs.nx)+0.5 {
				x = float64(fs.nx) + 0.5
			}
			i0 = (int(x) | int(x))
			i1 = i0 + 1

			if y < 0.5 {
				y = 0.5
			}
			if y > float64(fs.ny)+0.5 {
				y = float64(fs.ny) + 0.5
			}
			j0 = (int(y) | int(y))
			j1 = j0 + 1
			s1 = x - float64(i0)
			s0 = 1 - s1
			t1 = y - float64(j0)
			t0 = 1 - t1

			d[fs.idx(i, j)] = s0*(t0*d0[fs.idx(i0, j0)]+t1*d0[fs.idx(i0, j1)]) +
				s1*(t0*d0[fs.idx(i1, j0)]+t1*d0[fs.idx(i1, j1)])
		}
	}
	fs.setBoundary(bound, d)
}

// setBoundary sets the boundary conditions.
func (fs *Solver) setBoundary(bound BoundaryType, x cell) {
	for i := 1; i <= fs.nx; i++ {
		if bound == BoundaryLeftRight {
			x[fs.idx(0, i)] = -x[fs.idx(1, i)]
			x[fs.idx(fs.nx+1, i)] = -x[fs.idx(fs.nx, i)]
		} else {
			x[fs.idx(0, i)] = x[fs.idx(1, i)]
			x[fs.idx(fs.nx+1, i)] = x[fs.idx(fs.nx, i)]
		}
	}

	for i := 1; i <= fs.ny; i++ {
		if bound == BoundaryTopBottom {
			x[fs.idx(i, 0)] = -x[fs.idx(i, 1)]
			x[fs.idx(i, fs.ny+1)] = -x[fs.idx(i, fs.ny)]
		} else {
			x[fs.idx(i, 0)] = x[fs.idx(i, 1)]
			x[fs.idx(i, fs.ny+1)] = x[fs.idx(i, fs.ny)]
		}
	}

	x[fs.idx(0, 0)] = 0.5 * (x[fs.idx(1, 0)] + x[fs.idx(0, 1)])
	x[fs.idx(0, fs.ny+1)] = 0.5 * (x[fs.idx(1, fs.ny+1)] + x[fs.idx(0, fs.ny)])
	x[fs.idx(fs.nx+1, 0)] = 0.5 * (x[fs.idx(fs.nx, 0)] + x[fs.idx(fs.nx+1, 1)])
	x[fs.idx(fs.nx+1, fs.ny+1)] = 0.5 * (x[fs.idx(fs.nx, fs.ny+1)] + x[fs.idx(fs.nx+1, fs.ny)])
}

// idx returns the cell's index (position).
func (fs *Solver) idx(i, j int) int {
	return i + (fs.ny+2)*j
}
