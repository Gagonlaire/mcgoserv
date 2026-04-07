package world

import (
	"fmt"
	"iter"
	"math"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	"github.com/google/uuid"
)

type EntityID = int32
type DimensionID = string

type World struct {
	Dimensions     map[DimensionID]*Dimension
	EntitiesByID   map[EntityID]entities.Entity
	EntitiesByUUID map[uuid.UUID]entities.Entity
	PlayersByID    map[EntityID]*entities.Player
	PlayersByUUID  map[uuid.UUID]*entities.Player
	DirtyEntities  []entities.Entity
	Time           int64
	DayTime        int64
	Day            int64
	NextTimeUpdate int64
	LastEntityID   EntityID
	Rules          GameRules
}

type GameRules struct {
	DoDaylightCycle bool
}

type GenSettings struct {
	Seed int64
}

func NewWorld() *World {
	world := &World{
		Dimensions: map[DimensionID]*Dimension{
			"minecraft:overworld": {
				Type:   DefaultDimensionsType["minecraft:overworld"],
				Chunks: make(map[uint64]*mc.Chunk),
			},
			"minecraft:the_nether": {
				Type:   DefaultDimensionsType["minecraft:the_nether"],
				Chunks: make(map[uint64]*mc.Chunk),
			},
			"minecraft:the_end": {
				Type:   DefaultDimensionsType["minecraft:the_end"],
				Chunks: make(map[uint64]*mc.Chunk),
			},
		},
		EntitiesByID:   make(map[EntityID]entities.Entity),
		EntitiesByUUID: make(map[uuid.UUID]entities.Entity),
		PlayersByID:    make(map[EntityID]*entities.Player),
		PlayersByUUID:  make(map[uuid.UUID]*entities.Player),
	}

	for _, dimension := range world.Dimensions {
		dimension.World = world
	}

	return world
}

func (w *World) GetNextEntityID() EntityID {
	w.LastEntityID++
	return w.LastEntityID
}

func (w *World) Dimension(dimensionID DimensionID) *Dimension {
	return w.Dimensions[dimensionID]
}

func (w *World) GetEntityDimension(e entities.Entity) *Dimension {
	return w.Dimensions[e.Base().DimensionID]
}

func (w *World) OnlinePlayersCount() int {
	return len(w.PlayersByID)
}

func (w *World) Players() []*entities.Player {
	players := make([]*entities.Player, 0, len(w.PlayersByID))
	for _, player := range w.PlayersByID {
		players = append(players, player)
	}
	return players
}

func (w *World) EnqueueDirty(e entities.Entity) {
	base := e.Base()
	if !base.InSyncQueue {
		base.InSyncQueue = true
		w.DirtyEntities = append(w.DirtyEntities, e)
	}
}

func (w *World) AddPlayer(player *entities.Player, dimensionID DimensionID) error {
	base := player.Base()
	if base.EntityID == 0 {
		base.EntityID = w.GetNextEntityID()
	}

	if _, ok := w.EntitiesByID[base.EntityID]; ok {
		return fmt.Errorf("entity id already used: %d", base.EntityID)
	}
	if _, ok := w.EntitiesByUUID[base.UUID]; ok {
		return fmt.Errorf("entity uuid already used: %s", base.UUID)
	}

	dimension := w.Dimension(dimensionID)
	chunkX, chunkZ := GetChunkPosition(base.Pos[0], base.Pos[2])

	dimension.GetChunk(chunkX, chunkZ).Entities[base.EntityID] = struct{}{}
	base.DimensionID = dimensionID
	w.EntitiesByID[base.EntityID] = player
	w.EntitiesByUUID[base.UUID] = player
	w.PlayersByID[base.EntityID] = player
	w.PlayersByUUID[base.UUID] = player

	return nil
}

func (w *World) RemoveEntityByUUID(entityUUID uuid.UUID) {
	entity := w.EntitiesByUUID[entityUUID]
	if entity == nil {
		return
	}

	base := entity.Base()
	entityID := base.EntityID
	dimension := w.GetEntityDimension(entity)
	chunkX, chunkZ := GetChunkPosition(base.Pos[0], base.Pos[2])
	delete(dimension.GetChunk(chunkX, chunkZ).Entities, entityID)

	if player := w.PlayersByID[entityID]; player != nil {
		w.removePlayerWatchers(player)
		delete(w.PlayersByID, entityID)
		delete(w.PlayersByUUID, entityUUID)
	}

	base.DimensionID = ""
	delete(w.EntitiesByID, entityID)
	delete(w.EntitiesByUUID, entityUUID)
}

func (w *World) UpdateEntityChunk(entityID EntityID, oldX, oldZ, newX, newZ float64) {
	entity := w.EntitiesByID[entityID]
	if entity == nil {
		return
	}
	dimension := w.GetEntityDimension(entity)

	oldChunkX, oldChunkZ := GetChunkPosition(oldX, oldZ)
	newChunkX, newChunkZ := GetChunkPosition(newX, newZ)
	if oldChunkX == newChunkX && oldChunkZ == newChunkZ {
		return
	}

	delete(dimension.GetChunk(oldChunkX, oldChunkZ).Entities, entityID)
	dimension.GetChunk(newChunkX, newChunkZ).Entities[entityID] = struct{}{}
}

func (w *World) PlayersInChunkRadius(dimensionID DimensionID, centerChunkX, centerChunkZ, radius int) iter.Seq[*entities.Player] {
	return func(yield func(*entities.Player) bool) {
		dimension := w.Dimension(dimensionID)
		for x := centerChunkX - radius; x <= centerChunkX+radius; x++ {
			for z := centerChunkZ - radius; z <= centerChunkZ+radius; z++ {
				chunk := dimension.GetChunk(x, z)
				for entityID := range chunk.Entities {
					if player := w.PlayersByID[entityID]; player != nil {
						if !yield(player) {
							return
						}
					}
				}
			}
		}
	}
}

func (w *World) removePlayerWatchers(player *entities.Player) {
	dimension := w.GetEntityDimension(player)

	for pos := range player.Movement.VisibleChunks {
		delete(dimension.GetChunk(pos.X, pos.Z).Watchers, player.EntityID)
	}
	clear(player.Movement.VisibleChunks)
}

func GetChunkPosition(x, z float64) (int, int) {
	chunkX := int(math.Floor(x / 16.0))
	chunkZ := int(math.Floor(z / 16.0))
	return chunkX, chunkZ
}
