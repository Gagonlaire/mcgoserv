package world

// HeightmapType https://minecraft.wiki/w/Java_Edition_protocol/Chunk_format#Heightmap_structure.
type HeightmapType int32

const (
	HeightmapWorldSurface           HeightmapType = 1
	HeightmapMotionBlocking         HeightmapType = 4
	HeightmapMotionBlockingNoLeaves HeightmapType = 5
)

// Heightmaps TODO: wire real predicates (WORLD_SURFACE / MOTION_BLOCKING) once block registry exposes motion/solid flags.
// TODO: per-section heightmap dirty bitmap instead of chunk-wide bool.
type Heightmaps struct {
	values map[HeightmapType][]int32
}

// Values returns the cached heightmap values keyed by type, or nil if not
// computed. Each slice has 256 entries indexed by (z<<4)|x.
func (h *Heightmaps) Values() map[HeightmapType][]int32 { return h.values }

func (h *Heightmaps) recompute(_ *Chunk) {
	// TODO: mirror vanilla implementation
}
