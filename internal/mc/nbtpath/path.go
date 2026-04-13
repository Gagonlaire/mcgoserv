package nbtpath

import "github.com/Tnze/go-mc/nbt"

type Node struct {
	Name    string
	Index   int
	IsMatch bool
	Filter  nbt.StringifiedMessage
}

type Path struct {
	Nodes []Node
	Raw   string
}
