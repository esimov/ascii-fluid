package fluid

// Particle defines the general components of the particle system.
type Particle struct {
	x, y   float64
	vx, vy float64
	age    float64
	dead   bool
}

// NewParticle spawns a new particle at coordinates defined by {x, y}.
func NewParticle(x, y float64) *Particle {
	return &Particle{x: x, y: y}
}

// GetX retrieve the particle value at {x} position.
func (p *Particle) GetX() float64 {
	return p.x
}

// SetX set the particle value at {x} position.
func (p *Particle) SetX(val float64) {
	p.x = val
}

// GetY retrieve the particle value at {y} position.
func (p *Particle) GetY() float64 {
	return p.y
}

// SetY set the particle value at {y} position.
func (p *Particle) SetY(val float64) {
	p.y = val
}

// GetVx get the particle velocity at {x} position.
func (p *Particle) GetVx() float64 {
	return p.vx
}

// SetVx set the particle velocity at {x} position.
func (p *Particle) SetVx(val float64) {
	p.vx = val
}

// GetVy get the particle velocity at {y} position.
func (p *Particle) GetVy() float64 {
	return p.vy
}

// SetVy set the particle velocity at {y} position.
func (p *Particle) SetVy(val float64) {
	p.vy = val
}

// GetAge get the particle age.
func (p *Particle) GetAge() float64 {
	return p.age
}

// SetAge set the particle age.
func (p *Particle) SetAge(age float64) {
	p.age = age
}

// GetDeath check if a particle is dead.
func (p *Particle) GetDeath() bool {
	return p.dead
}

// SetDeath set a particle as death.
func (p *Particle) SetDeath(dead bool) {
	p.dead = dead
}
