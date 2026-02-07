package world

type World struct {
	Dimensions     map[string]*Dimension
	Rules          GameRules
	Time           int64
	DayTime        int64
	Day            int64
	NextTimeUpdate int64
	// todo: add connection to players when switching to play state
	// Players map[uuid.UUID]*Player ?
}

type GameRules struct {
	DoDaylightCycle bool
}

type GenSettings struct {
	Seed int64
}

func NewWorld() *World {
	world := &World{
		Dimensions: map[string]*Dimension{
			"minecraft:overworld": {
				Type: DefaultDimensionsType["minecraft:overworld"],
			},
			"minecraft:the_nether": {
				Type: DefaultDimensionsType["minecraft:the_nether"],
			},
			"minecraft:the_end": {
				Type: DefaultDimensionsType["minecraft:the_end"],
			},
		},
	}

	for _, v := range world.Dimensions {
		v.World = world
	}

	return world
}
