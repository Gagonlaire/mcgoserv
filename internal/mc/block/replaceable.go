package block

// NOTE: temporary link with `internal/mcdata/data/minecraft/tags/block/replaceable.json`.
// TODO: find a way to use the game registry data for this and other game behaviors using json files
var replaceableNames = [...]string{
	"air",
	"water",
	"lava",
	"short_grass",
	"fern",
	"dead_bush",
	"bush",
	"short_dry_grass",
	"tall_dry_grass",
	"seagrass",
	"tall_seagrass",
	"fire",
	"soul_fire",
	"snow",
	"vine",
	"glow_lichen",
	"resin_clump",
	"light",
	"tall_grass",
	"large_fern",
	"structure_void",
	"void_air",
	"cave_air",
	"bubble_column",
	"warped_roots",
	"nether_sprouts",
	"crimson_roots",
	"leaf_litter",
	"hanging_roots",
}

var replaceableIDs [len(registry)]bool

func init() {
	for _, name := range replaceableNames {
		if id, ok := FromString(name); ok {
			replaceableIDs[id] = true
		}
	}
}

// IsReplaceableState reports whether a block at the given state ID can be overwritten by a placement
func IsReplaceableState(stateID int32) bool {
	id, ok := FromStateID(int(stateID))
	if !ok {
		return false
	}
	return replaceableIDs[id]
}

// IsSolidFullCube reports whether a block at the given state ID occupies a full cubic collision volume
// TODO: replace with real per-state collision shape data when the gen pipeline surfaces blocks.json shape info.
func IsSolidFullCube(stateID int32) bool {
	id, ok := FromStateID(int(stateID))
	if !ok {
		return false
	}
	p := id.Raw()
	return p.BoundingBox == "block" && !p.Transparent
}
