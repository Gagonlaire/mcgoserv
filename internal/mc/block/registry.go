package block

// TODO: when all block implemented, remove related checks
var behaviorByID [len(registry)]Behavior

// Register installs a Behavior against an ID. Should be called by RegisterAll or by block family
func Register(id ID, b Behavior) {
	if int(id) < 0 || int(id) >= len(behaviorByID) {
		panic("block.Register: id out of range")
	}
	if behaviorByID[id] != nil {
		panic("block.Register: id already registered")
	}
	behaviorByID[id] = b
}

func Lookup(id ID) (Behavior, bool) {
	if int(id) < 0 || int(id) >= len(behaviorByID) {
		return nil, false
	}
	b := behaviorByID[id]
	if b == nil {
		return nil, false
	}
	return b, true
}
