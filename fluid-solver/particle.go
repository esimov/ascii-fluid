package fluid

type Particle struct {
	x, y   int
	vx, vy float64
	age    int
	dead   bool
}

func NewParticle(x, y int) *Particle {
	return &Particle{x: x, y: y}
}

func (p *Particle) GetX() float64 {
	return p.vx
}

func (p *Particle) SetX(val float64) {
	p.vx = val
}

func (p *Particle) GetY() float64 {
	return p.vy
}

func (p *Particle) SetY(val float64) {
	p.vy = val
}
