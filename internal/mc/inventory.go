package mc

import "fmt"

const (
	SlotHotbarStart = 36
	SlotHotbarEnd   = 44
	SlotOffHand     = 45
)

type PlayerInventory struct {
	Slots [46]Slot
}

func NewPlayerInventory() *PlayerInventory {
	return &PlayerInventory{}
}

func HotbarToInternal(hotbarSlot int) int {
	if hotbarSlot < 0 || hotbarSlot > 8 {
		return -1
	}
	return SlotHotbarStart + hotbarSlot
}

func InternalToHotbar(internalIndex int) int {
	if internalIndex >= SlotHotbarStart && internalIndex <= SlotHotbarEnd {
		return internalIndex - SlotHotbarStart
	}
	return -1
}

func (inv *PlayerInventory) Set(index int, s Slot) error {
	if index < 0 || index >= len(inv.Slots) {
		return fmt.Errorf("index %d out of bounds", index)
	}
	inv.Slots[index] = s
	return nil
}

func (inv *PlayerInventory) Get(index int) Slot {
	if index < 0 || index >= len(inv.Slots) {
		return Slot{}
	}
	return inv.Slots[index]
}

func (inv *PlayerInventory) GetHotbarItem(selectedSlot int) Slot {
	idx := HotbarToInternal(selectedSlot)
	if idx == -1 {
		return Slot{}
	}
	return inv.Slots[idx]
}
