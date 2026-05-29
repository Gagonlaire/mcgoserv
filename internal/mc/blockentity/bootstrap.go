package blockentity

// RegisterAll mirrors block.RegisterAll.
func RegisterAll() {
	Register(TypeSign, newSignBE)
}
