package fluid

import "math"

type cell []float64

type fluidSolver struct {
	n           int
	dt          float64
	diffusion   float64
	viscosity   float64
	iterations  int
	doVorticity bool
	doBouyancy  bool
	numOfCells  int

	u cell
	v cell
	d cell

	uOld cell
	vOld cell
	dOld cell

	curlData cell
}

type BoundaryType int

const (
	BoundaryNone BoundaryType = iota
	BoundaryLeftRight
	BoundaryTopBottom
)

func NewSolver(n int) *fluidSolver {
	fs := &fluidSolver{
		n:           n,
		dt:          0.1,
		diffusion:   0.0002,
		viscosity:   0.0,
		iterations:  10,
		doVorticity: true,
		doBouyancy:  true,
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

func (fs *fluidSolver) idx(i, j int) int {
	return i + (fs.n+2)*j
}

func (fs *fluidSolver) SetCell(cellType interface{}, x, y int, val float64) {
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

func (fs *fluidSolver) GetCell(cellType interface{}, x, y int) (result float64) {
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

func (fs *fluidSolver) DensityStep() {
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

func (fs *fluidSolver) VelocityStep() {
	fs.addSource(fs.u, fs.uOld)
	fs.addSource(fs.v, fs.vOld)

	if fs.doVorticity {
		fs.calcVorticityConfinement(fs.uOld, fs.vOld)
		fs.addSource(fs.u, fs.uOld)
		fs.addSource(fs.v, fs.vOld)
	}

	if fs.doBouyancy {
		fs.buoyancy(fs.vOld)
		fs.addSource(fs.v, fs.vOld)
	}

	fs.swapU()
	fs.diffuse(BoundaryLeftRight, fs.u, fs.uOld, fs.viscosity)

	fs.swapV()
	fs.diffuse(BoundaryLeftRight, fs.v, fs.vOld, fs.viscosity)

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

func (fs *fluidSolver) ResetDensity() {
	for i := 0; i < fs.numOfCells; i++ {
		fs.d[i] = 0
	}
}

func (fs *fluidSolver) ResetVelocity() {
	for i := 0; i < fs.numOfCells; i++ {
		// Set a small value so we can render the velocity field
		fs.v[i] = 0.001
		fs.u[i] = 0.001
	}
}

func (fs *fluidSolver) swapU() {
	tmp := fs.u
	fs.u = fs.uOld
	fs.uOld = tmp
}

func (fs *fluidSolver) swapV() {
	tmp := fs.v
	fs.v = fs.vOld
	fs.vOld = tmp
}

func (fs *fluidSolver) swapD() {
	tmp := fs.d
	fs.d = fs.dOld
	fs.dOld = tmp
}

func (fs *fluidSolver) addSource(x, s cell) {
	for i := 0; i < fs.numOfCells; i++ {
		x[i] += s[i] * fs.dt
	}
}

func (fs *fluidSolver) curl(i, j int) float64 {
	duDy := fs.u[fs.idx(i, j+1)] - fs.u[fs.idx(i, j-1)]*0.5
	dvDx := fs.v[fs.idx(i+1, j)] - fs.v[fs.idx(i-1, j)]*0.5

	return duDy - dvDx
}

func (fs *fluidSolver) calcVorticityConfinement(x, y cell) {
	var (
		i, j            int
		dx, dy, norm, v float64
	)

	for i = 1; i <= fs.n; i++ {
		for j = 1; j <= fs.n; j++ {
			fs.curlData[fs.idx(i, j)] = math.Abs(fs.curl(i, j))
		}
	}

	for i = 1; i <= fs.n; i++ {
		for j = 1; j <= fs.n; j++ {
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

func (fs *fluidSolver) buoyancy(buoy cell) cell {
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
	tAmb /= float64(fs.n * fs.n)

	// For each cell compute the bouyancy force
	for i = 1; i <= fs.n; i++ {
		for j = 1; j <= fs.n; j++ {
			buoy[fs.idx(i, j)] = a*fs.d[fs.idx(i, j)] + -b*(fs.d[fs.idx(i, j)]-tAmb)
		}
	}
	return buoy
}

func (fs *fluidSolver) diffuse(bound BoundaryType, x, x0 cell, diffusion float64) {
	a := fs.dt * diffusion * float64(fs.n*fs.n)
	fs.linearSolve(bound, x, x0, a, 1.0+4.0*a)
}

func (fs *fluidSolver) linearSolve(bound BoundaryType, x, x0 cell, a, c float64) {
	invC := 1.0 / c

	for k := 0; k < fs.iterations; k++ {
		for i := 1; i <= fs.n; i++ {
			for j := 1; j <= fs.n; j++ {
				x[fs.idx(i, j)] = (x0[fs.idx(i, j)] + a*(x[fs.idx(i-1, j)]+x[fs.idx(i+1, j)]+x[fs.idx(i, j-1)]+x[fs.idx(i, j+1)])) * invC
			}
		}
	}
	fs.setBoundary(bound, x)
}

func (fs *fluidSolver) project(u, v, p, div cell) {
	// Calculate the gradient field
	h := 1.0 / float64(fs.n)
	for i := 1; i <= fs.n; i++ {
		for j := 1; j <= fs.n; j++ {
			div[fs.idx(i, j)] = -0.5 * h * (u[fs.idx(i+1, j)] - u[fs.idx(i-1, j)] + v[fs.idx(i, j+1)] - v[fs.idx(i, j-1)])
			p[fs.idx(i, j)] = 0
		}
	}
	fs.setBoundary(BoundaryNone, div)
	fs.setBoundary(BoundaryNone, p)

	// Solve the Poisson equation
	fs.linearSolve(BoundaryNone, p, div, 1, 4)

	// Substract the gradient field from the velocity field to get the mass conserving velocity field.
	for i := 1; i <= fs.n; i++ {
		for j := 1; j <= fs.n; j++ {
			u[fs.idx(i, j)] = 0.5 * (p[fs.idx(i+1, j)] - p[fs.idx(i-1, j)]) / h
			v[fs.idx(i, j)] = 0.5 * (p[fs.idx(i, j+1)] - p[fs.idx(i, j-1)]) / h
		}
	}
	fs.setBoundary(BoundaryLeftRight, u)
	fs.setBoundary(BoundaryTopBottom, v)
}

func (fs *fluidSolver) advect(bound BoundaryType, d, d0, u, v cell) {
	var (
		i, j, i0, j0, i1, j1      int
		x, y, s0, t0, s1, t1, dt0 float64
	)
	dt0 = fs.dt * float64(fs.n)

	for i = 1; i <= fs.n; i++ {
		for j = 1; j <= fs.n; j++ {
			x = float64(i) - dt0*u[fs.idx(i, j)]
			y = float64(j) - dt0*v[fs.idx(i, j)]
			if x < 0.5 {
				x = 0.5
			}
			if x > float64(fs.n)+0.5 {
				x = float64(fs.n) + 0.5
			}
			i0 = (int(x) | int(x))
			i1 = i0 + 1

			if y < 0.5 {
				y = 0.5
			}
			if y > float64(fs.n)+0.5 {
				y = float64(fs.n) + 0.5
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

func (fs *fluidSolver) setBoundary(bound BoundaryType, x cell) {
	for i := 1; i <= fs.n; i++ {
		if bound == BoundaryLeftRight {
			x[fs.idx(0, i)] = -x[fs.idx(1, i)]
			x[fs.idx(fs.n, i)] = -x[fs.idx(fs.n-1, i)]
		} else {
			x[fs.idx(0, i)] = x[fs.idx(1, i)]
			x[fs.idx(fs.n, i)] = x[fs.idx(fs.n-1, i)]
		}
		if bound == BoundaryTopBottom {
			x[fs.idx(i, 0)] = -x[fs.idx(i, 1)]
			x[fs.idx(i, fs.n)] = -x[fs.idx(i, fs.n-1)]
		} else {
			x[fs.idx(i, 0)] = x[fs.idx(i, 1)]
			x[fs.idx(i, fs.n)] = x[fs.idx(i, fs.n-1)]
		}
	}

	x[fs.idx(0, 0)] = 0.5 * (x[fs.idx(1, 0)] + x[fs.idx(0, 1)])
	x[fs.idx(0, fs.n+1)] = 0.5 * (x[fs.idx(1, fs.n+1)] + x[fs.idx(0, fs.n)])
	x[fs.idx(fs.n+1, 0)] = 0.5 * (x[fs.idx(fs.n, 0)] + x[fs.idx(fs.n+1, 1)])
	x[fs.idx(fs.n+1, fs.n+1)] = 0.5 * (x[fs.idx(fs.n, fs.n+1)] + x[fs.idx(fs.n+1, fs.n)])
}
