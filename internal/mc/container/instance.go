package container

import (
	"fmt"

	"github.com/Gagonlaire/mcgoserv/internal/mc/item"
)

type Instance struct {
	Slots []Slot
}

func NewInstance(size int) Instance {
	return Instance{Slots: make([]Slot, size)}
}

func (inv *Instance) Get(index int) Slot {
	if index < 0 || index >= len(inv.Slots) {
		return Slot{}
	}
	return inv.Slots[index]
}

func (inv *Instance) Set(index int, s Slot) error {
	if index < 0 || index >= len(inv.Slots) {
		return fmt.Errorf("index %d out of bounds", index)
	}
	inv.Slots[index] = s
	return nil
}

func (inv *Instance) Clear() {
	for i := range inv.Slots {
		inv.Slots[i] = Slot{}
	}
}

func (inv *Instance) FirstEmpty() int {
	for i := range inv.Slots {
		if inv.Slots[i].Count <= 0 {
			return i
		}
	}
	return -1
}

func (inv *Instance) AddItem(stack Slot) Slot {
	return inv.addItemRange(stack, 0, len(inv.Slots)-1)
}

func (inv *Instance) addItemRange(stack Slot, start, end int) Slot {
	if stack.Count <= 0 {
		return stack
	}
	if start < 0 {
		start = 0
	}
	if end >= len(inv.Slots) {
		end = len(inv.Slots) - 1
	}

	max := int32(item.ID(stack.ItemID).StackSize())
	if max <= 0 {
		max = 1
	}

	for i := start; i <= end && stack.Count > 0; i++ {
		existing := inv.Slots[i]
		if !existing.StacksWith(stack) {
			continue
		}
		room := max - existing.Count
		if room <= 0 {
			continue
		}
		take := room
		if take > stack.Count {
			take = stack.Count
		}
		existing.Count += take
		inv.Slots[i] = existing
		stack.Count -= take
	}

	for i := start; i <= end && stack.Count > 0; i++ {
		if inv.Slots[i].Count > 0 {
			continue
		}
		take := stack.Count
		if take > max {
			take = max
		}
		placed := stack
		placed.Count = take
		inv.Slots[i] = placed
		stack.Count -= take
	}

	return stack
}
