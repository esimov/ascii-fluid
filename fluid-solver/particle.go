package fluid

type Particle struct {
	x, y   float64
	vx, vy float64
	age    float64
	dead   bool
}

func NewParticle(x, y float64) *Particle {
	return &Particle{x: x, y: y}
}

func (p *Particle) GetX() int {
	return int(p.x)
}

func (p *Particle) SetX(val float64) {
	p.x = val
}

func (p *Particle) GetY() int {
	return int(p.y)
}

func (p *Particle) SetY(val float64) {
	p.y = val
}

func (p *Particle) GetVx() int {
	return int(p.vx)
}

func (p *Particle) SetVx(val float64) {
	p.vx = val
}

func (p *Particle) GetVy() int {
	return int(p.vy)
}

func (p *Particle) SetVy(val float64) {
	p.vy = val
}

func (p *Particle) GetAge() float64 {
	return p.age
}

func (p *Particle) SetAge(age float64) {
	p.age = age
}

func (p *Particle) GetDeath() bool {
	return p.dead
}

func (p *Particle) SetDeath(dead bool) {
	p.dead = dead
}
