package world

type Dimension struct {
	World *World
	Type  DimensionType

	// Add chunks and entities
}

type DimensionType struct {
	CoordinateScale float64
	HasSkylight     bool
	HasCeiling      bool
	AmbientLight    float32
	HasFixedTime    bool
}

func (d Dimension) tick() {

}

var (
	DefaultDimensionsType = map[string]DimensionType{
		"minecraft:overworld": {
			CoordinateScale: 1.0,
			HasSkylight:     true,
			HasCeiling:      false,
			AmbientLight:    0.0,
			HasFixedTime:    false,
		},
		"minecraft:the_nether": {
			CoordinateScale: 8.0,
			HasSkylight:     false,
			HasCeiling:      true,
			AmbientLight:    0.0,
			HasFixedTime:    true,
		},
		"minecraft:the_end": {
			CoordinateScale: 1.0,
			HasSkylight:     false,
			HasCeiling:      false,
			AmbientLight:    0.0,
			HasFixedTime:    true,
		},
	}
)
