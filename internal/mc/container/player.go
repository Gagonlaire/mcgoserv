package container

type ArmorSlot int

const (
	ArmorHelmet ArmorSlot = iota
	ArmorChestplate
	ArmorLeggings
	ArmorBoots
)

func (a ArmorSlot) internal() int {
	return SlotArmorStart + int(a)
}

// PlayerInventory is the 46-slot Notchian player inventory:
// crafting result + 2x2 crafting grid, armor, main, hotbar, off-hand.
type PlayerInventory struct {
	Instance
	SelectedHotbar int32
}

func NewPlayerInventory() *PlayerInventory {
	return &PlayerInventory{Instance: NewInstance(PlayerInventorySize)}
}

func (p *PlayerInventory) Held() Slot {
	return p.Get(HotbarToInternal(int(p.SelectedHotbar)))
}

func (p *PlayerInventory) SetHeld(s Slot) error {
	return p.Set(HotbarToInternal(int(p.SelectedHotbar)), s)
}

func (p *PlayerInventory) Armor(a ArmorSlot) Slot {
	return p.Get(a.internal())
}

func (p *PlayerInventory) SetArmor(a ArmorSlot, s Slot) error {
	return p.Set(a.internal(), s)
}

func (p *PlayerInventory) Offhand() Slot {
	return p.Get(SlotOffHand)
}

func (p *PlayerInventory) SetOffhand(s Slot) error {
	return p.Set(SlotOffHand, s)
}

// AddItem restricts pickup to main + hotbar slots (9..44), matching vanilla: armor, off-hand,
// and crafting slots never receive items via pickup.
func (p *PlayerInventory) AddItem(stack Slot) Slot {
	return p.addItemRange(stack, SlotMainStart, SlotHotbarEnd)
}
