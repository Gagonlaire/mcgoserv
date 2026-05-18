package world

import (
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	"github.com/google/uuid"
)

// makeEntity builds a *Zombie and overrides its type ID
func makeEntity(typeID entity.ID, pos [3]float64) *entity.Zombie {
	z := entity.NewZombie("minecraft:overworld", pos, [2]float32{0, 0})
	z.ID = typeID
	return z
}

func addEntity(w *World, e entity.Entity) {
	id := w.GetNextEntityID()
	base := e.Base()
	base.EntityID = id
	w.EntitiesByID[id] = e
	w.EntitiesByUUID[uuid.UUID(base.UUID)] = e
}

func TestResolveSelectorType(t *testing.T) {
	t.Run("positive_filters_to_matching_type", func(t *testing.T) {
		w := NewWorld()
		zombie := makeEntity(entity.ZombieID, [3]float64{0, 0, 0})
		creeper := makeEntity(entity.CreeperID, [3]float64{1, 0, 0})
		addEntity(w, zombie)
		addEntity(w, creeper)

		sel := &mc.Selector{
			Variable:    mc.SelectorVariableAllEntities,
			TypeInclude: mc.Optional[string]{Value: "zombie", Present: true},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != zombie {
			t.Fatalf("want only zombie, got %v", got)
		}
	})

	t.Run("negation_excludes_matching_type", func(t *testing.T) {
		w := NewWorld()
		zombie := makeEntity(entity.ZombieID, [3]float64{0, 0, 0})
		creeper := makeEntity(entity.CreeperID, [3]float64{1, 0, 0})
		addEntity(w, zombie)
		addEntity(w, creeper)

		sel := &mc.Selector{
			Variable:    mc.SelectorVariableAllEntities,
			TypeExclude: []string{"zombie"},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != creeper {
			t.Fatalf("want only creeper, got %v", got)
		}
	})

	t.Run("multi_negation_excludes_all", func(t *testing.T) {
		w := NewWorld()
		zombie := makeEntity(entity.ZombieID, [3]float64{0, 0, 0})
		creeper := makeEntity(entity.CreeperID, [3]float64{1, 0, 0})
		pig := makeEntity(entity.PigID, [3]float64{2, 0, 0})
		addEntity(w, zombie)
		addEntity(w, creeper)
		addEntity(w, pig)

		sel := &mc.Selector{
			Variable:    mc.SelectorVariableAllEntities,
			TypeExclude: []string{"zombie", "creeper"},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != pig {
			t.Fatalf("want only pig, got %v", got)
		}
	})

	t.Run("nearest_entity_picks_nearest_after_filter", func(t *testing.T) {
		w := NewWorld()
		nearCow := makeEntity(entity.CowID, [3]float64{1, 0, 0})
		nearerZombie := makeEntity(entity.ZombieID, [3]float64{5, 0, 0})
		farZombie := makeEntity(entity.ZombieID, [3]float64{10, 0, 0})
		addEntity(w, nearCow)
		addEntity(w, nearerZombie)
		addEntity(w, farZombie)

		sel := &mc.Selector{
			Variable:    mc.SelectorVariableNearestEntity,
			TypeInclude: mc.Optional[string]{Value: "zombie", Present: true},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != nearerZombie {
			t.Fatalf("want nearer zombie, got %v", got)
		}
	})

	t.Run("self_with_nonmatching_type_returns_empty", func(t *testing.T) {
		w := NewWorld()
		zombie := makeEntity(entity.ZombieID, [3]float64{0, 0, 0})
		addEntity(w, zombie)

		sel := &mc.Selector{
			Variable:    mc.SelectorVariableSelf,
			TypeInclude: mc.Optional[string]{Value: "creeper", Present: true},
		}
		got := w.resolveSelector(sel, uuid.UUID(zombie.UUID), [3]float64{})
		if len(got) != 0 {
			t.Fatalf("want empty (filter mismatches self), got %v", got)
		}
	})
}
