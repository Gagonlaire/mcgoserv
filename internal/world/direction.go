package world

import "math"

type Direction uint8

const (
	DirectionDown Direction = iota
	DirectionUp
	DirectionNorth
	DirectionSouth
	DirectionWest
	DirectionEast
)

func (d Direction) Vector() (dx, dy, dz int) {
	switch d {
	case DirectionDown:
		return 0, -1, 0
	case DirectionUp:
		return 0, 1, 0
	case DirectionNorth:
		return 0, 0, -1
	case DirectionSouth:
		return 0, 0, 1
	case DirectionWest:
		return -1, 0, 0
	case DirectionEast:
		return 1, 0, 0
	}
	return 0, 0, 0
}

func (d Direction) Opposite() Direction {
	switch d {
	case DirectionDown:
		return DirectionUp
	case DirectionUp:
		return DirectionDown
	case DirectionNorth:
		return DirectionSouth
	case DirectionSouth:
		return DirectionNorth
	case DirectionWest:
		return DirectionEast
	case DirectionEast:
		return DirectionWest
	}
	return d
}

func (d Direction) IsCardinal() bool {
	return d >= DirectionNorth
}

func (d Direction) Offset(p BlockPos) BlockPos {
	return p.Offset(d)
}

// RotateCCW returns the next cardinal direction rotating counter-clockwise
// when viewed from above (Mojang convention): N→W→S→E→N.
func (d Direction) RotateCCW() Direction {
	switch d {
	case DirectionNorth:
		return DirectionWest
	case DirectionWest:
		return DirectionSouth
	case DirectionSouth:
		return DirectionEast
	case DirectionEast:
		return DirectionNorth
	}
	return d
}

// RotateCW returns the next cardinal direction rotating clockwise when viewed
// from above (Mojang convention): N→E→S→W→N.
func (d Direction) RotateCW() Direction {
	switch d {
	case DirectionNorth:
		return DirectionEast
	case DirectionEast:
		return DirectionSouth
	case DirectionSouth:
		return DirectionWest
	case DirectionWest:
		return DirectionNorth
	}
	return d
}

func CardinalFromYaw(yaw float32) Direction {
	y := math.Mod(float64(yaw), 360)
	if y < 0 {
		y += 360
	}
	switch int((y+45)/90) % 4 {
	case 0:
		return DirectionSouth
	case 1:
		return DirectionWest
	case 2:
		return DirectionNorth
	case 3:
		return DirectionEast
	}
	return DirectionSouth
}
