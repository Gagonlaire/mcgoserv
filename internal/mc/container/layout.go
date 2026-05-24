package container

const (
	PlayerInventorySize = 46
	SlotCraftResult     = 0
	SlotCraftStart      = 1
	SlotCraftEnd        = 4
	SlotArmorStart      = 5
	SlotArmorEnd        = 8
	SlotMainStart       = 9
	SlotMainEnd         = 35
	SlotHotbarStart     = 36
	SlotHotbarEnd       = 44
	SlotOffHand         = 45
)

// HotbarToInternal maps a hotbar index (0..8) to its internal player-inventory slot, or -1 if out of range.
func HotbarToInternal(hotbarSlot int) int {
	if hotbarSlot < 0 || hotbarSlot > 8 {
		return -1
	}
	return SlotHotbarStart + hotbarSlot
}

// InternalToHotbar maps an internal slot back to a hotbar index (0..8), or -1 if the slot is outside the hotbar.
func InternalToHotbar(internalIndex int) int {
	if internalIndex >= SlotHotbarStart && internalIndex <= SlotHotbarEnd {
		return internalIndex - SlotHotbarStart
	}
	return -1
}
