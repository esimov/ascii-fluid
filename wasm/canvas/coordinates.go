package canvas

type detection struct {
	X int `json:"row"`
	Y int `json:"col"`
}

func (c *Canvas) newDetection(x, y int) *detection {
	return &detection{
		X: x,
		Y: y,
	}
}
