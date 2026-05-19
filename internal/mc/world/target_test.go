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

func makeEntityWithRot(typeID entity.ID, pos [3]float64, yaw, pitch float32) *entity.Zombie {
	z := entity.NewZombie("minecraft:overworld", pos, [2]float32{yaw, pitch})
	z.ID = typeID
	return z
}

func makePlayer(name string, pos [3]float64, xpLevel, gameMode int32) *entity.Player {
	p := &entity.Player{Name: name, XpLevel: xpLevel, GameMode: gameMode}
	p.BaseEntity.ID = entity.PlayerID
	p.BaseEntity.Position = pos
	p.BaseEntity.UUID = entity.NbtUUID(uuid.New())
	return p
}

func addEntity(w *World, e entity.Entity) {
	id := w.GetNextEntityID()
	base := e.Base()
	base.EntityID = id
	w.EntitiesByID[id] = e
	w.EntitiesByUUID[uuid.UUID(base.UUID)] = e
	if p, ok := e.(*entity.Player); ok {
		w.PlayersByID[id] = p
		w.PlayersByUUID[uuid.UUID(base.UUID)] = p
	}
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

func TestResolveSelectorNbt(t *testing.T) {
	t.Run("subset_match_on_isbaby_field", func(t *testing.T) {
		w := NewWorld()
		baby := makeEntity(entity.ZombieID, [3]float64{0, 0, 0})
		baby.IsBaby = true
		adult := makeEntity(entity.ZombieID, [3]float64{1, 0, 0})
		adult.IsBaby = false
		addEntity(w, baby)
		addEntity(w, adult)

		sel := &mc.Selector{
			Variable:    mc.SelectorVariableAllEntities,
			NbtIncludes: []any{map[string]any{"IsBaby": int8(1)}},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != baby {
			t.Fatalf("want only baby, got %v", got)
		}
	})

	t.Run("negation_excludes_matching", func(t *testing.T) {
		w := NewWorld()
		baby := makeEntity(entity.ZombieID, [3]float64{0, 0, 0})
		baby.IsBaby = true
		adult := makeEntity(entity.ZombieID, [3]float64{1, 0, 0})
		adult.IsBaby = false
		addEntity(w, baby)
		addEntity(w, adult)

		sel := &mc.Selector{
			Variable:    mc.SelectorVariableAllEntities,
			NbtExcludes: []any{map[string]any{"IsBaby": int8(1)}},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != adult {
			t.Fatalf("want only adult, got %v", got)
		}
	})

	t.Run("multiple_positives_require_all_to_match", func(t *testing.T) {
		w := NewWorld()
		zombie := makeEntity(entity.ZombieID, [3]float64{0, 0, 0})
		zombie.IsBaby = true
		zombie.CanBreakDoors = true
		addEntity(w, zombie)
		other := makeEntity(entity.ZombieID, [3]float64{1, 0, 0})
		other.IsBaby = true
		other.CanBreakDoors = false
		addEntity(w, other)

		sel := &mc.Selector{
			Variable: mc.SelectorVariableAllEntities,
			NbtIncludes: []any{
				map[string]any{"IsBaby": int8(1)},
				map[string]any{"CanBreakDoors": int8(1)},
			},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != zombie {
			t.Fatalf("want only door-breaking baby, got %v", got)
		}
	})

	t.Run("missing_field_does_not_match", func(t *testing.T) {
		w := NewWorld()
		zombie := makeEntity(entity.ZombieID, [3]float64{0, 0, 0})
		addEntity(w, zombie)

		sel := &mc.Selector{
			Variable:    mc.SelectorVariableAllEntities,
			NbtIncludes: []any{map[string]any{"NoSuchField": int8(1)}},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 0 {
			t.Fatalf("want empty, got %v", got)
		}
	})
}

func TestResolveSelectorDistance(t *testing.T) {
	t.Run("upper_bound_excludes_far", func(t *testing.T) {
		w := NewWorld()
		near := makeEntity(entity.ZombieID, [3]float64{2, 0, 0})
		far := makeEntity(entity.ZombieID, [3]float64{10, 0, 0})
		addEntity(w, near)
		addEntity(w, far)

		sel := &mc.Selector{
			Variable: mc.SelectorVariableAllEntities,
			Distance: mc.Optional[mc.FloatRange]{
				Present: true,
				Value: mc.FloatRange{
					Max: mc.Optional[float64]{Value: 5, Present: true},
				},
			},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != near {
			t.Fatalf("want only near, got %v", got)
		}
	})

	t.Run("lower_bound_excludes_close", func(t *testing.T) {
		w := NewWorld()
		near := makeEntity(entity.ZombieID, [3]float64{2, 0, 0})
		far := makeEntity(entity.ZombieID, [3]float64{10, 0, 0})
		addEntity(w, near)
		addEntity(w, far)

		sel := &mc.Selector{
			Variable: mc.SelectorVariableAllEntities,
			Distance: mc.Optional[mc.FloatRange]{
				Present: true,
				Value: mc.FloatRange{
					Min: mc.Optional[float64]{Value: 5, Present: true},
				},
			},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != far {
			t.Fatalf("want only far, got %v", got)
		}
	})

	t.Run("xyz_overrides_reference_point", func(t *testing.T) {
		// Source at origin, but x=20 overrides. Entity at (22,0,0) is distance 2 from x=20.
		w := NewWorld()
		ent := makeEntity(entity.ZombieID, [3]float64{22, 0, 0})
		addEntity(w, ent)

		sel := &mc.Selector{
			Variable: mc.SelectorVariableAllEntities,
			X:        mc.Optional[float64]{Value: 20, Present: true},
			Distance: mc.Optional[mc.FloatRange]{
				Present: true,
				Value: mc.FloatRange{
					Max: mc.Optional[float64]{Value: 3, Present: true},
				},
			},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{0, 0, 0})
		if len(got) != 1 {
			t.Fatalf("want 1 entity (within 3 of x=20), got %d", len(got))
		}
	})
}

func TestResolveSelectorVolume(t *testing.T) {
	t.Run("entity_inside_box_matches", func(t *testing.T) {
		w := NewWorld()
		inside := makeEntity(entity.ZombieID, [3]float64{5, 5, 5})
		outside := makeEntity(entity.ZombieID, [3]float64{50, 0, 0})
		addEntity(w, inside)
		addEntity(w, outside)

		sel := &mc.Selector{
			Variable: mc.SelectorVariableAllEntities,
			Dx:       mc.Optional[float64]{Value: 10, Present: true},
			Dy:       mc.Optional[float64]{Value: 10, Present: true},
			Dz:       mc.Optional[float64]{Value: 10, Present: true},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != inside {
			t.Fatalf("want only inside, got %v", got)
		}
	})

	t.Run("negative_dx_normalizes_box", func(t *testing.T) {
		// Origin (10,0,0), dx=-5 → box X spans 5..10.
		w := NewWorld()
		inside := makeEntity(entity.ZombieID, [3]float64{7, 0, 0})
		outside := makeEntity(entity.ZombieID, [3]float64{15, 0, 0})
		addEntity(w, inside)
		addEntity(w, outside)

		sel := &mc.Selector{
			Variable: mc.SelectorVariableAllEntities,
			X:        mc.Optional[float64]{Value: 10, Present: true},
			Y:        mc.Optional[float64]{Value: 0, Present: true},
			Z:        mc.Optional[float64]{Value: 0, Present: true},
			Dx:       mc.Optional[float64]{Value: -5, Present: true},
			Dy:       mc.Optional[float64]{Value: 5, Present: true},
			Dz:       mc.Optional[float64]{Value: 5, Present: true},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != inside {
			t.Fatalf("want only inside (negative dx normalized), got %v", got)
		}
	})

	t.Run("entity_hitbox_grazes_edge", func(t *testing.T) {
		// Zombie width ~0.6 → half=0.3. Entity at (10.2,0,0), box X=[10..10].
		// Hitbox spans x=[9.9..10.5], overlaps box edge at x=10.
		w := NewWorld()
		grazing := makeEntity(entity.ZombieID, [3]float64{10.2, 0, 0})
		addEntity(w, grazing)

		sel := &mc.Selector{
			Variable: mc.SelectorVariableAllEntities,
			X:        mc.Optional[float64]{Value: 10, Present: true},
			Y:        mc.Optional[float64]{Value: 0, Present: true},
			Z:        mc.Optional[float64]{Value: 0, Present: true},
			Dx:       mc.Optional[float64]{Value: 0, Present: true},
			Dy:       mc.Optional[float64]{Value: 2, Present: true},
			Dz:       mc.Optional[float64]{Value: 0, Present: true},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 {
			t.Fatalf("want 1 (hitbox grazing edge), got %d", len(got))
		}
	})
}

func TestResolveSelectorRotation(t *testing.T) {
	t.Run("y_rotation_filters_by_yaw", func(t *testing.T) {
		w := NewWorld()
		south := makeEntityWithRot(entity.ZombieID, [3]float64{0, 0, 0}, 0, 0)
		east := makeEntityWithRot(entity.ZombieID, [3]float64{1, 0, 0}, -90, 0)
		addEntity(w, south)
		addEntity(w, east)

		sel := &mc.Selector{
			Variable: mc.SelectorVariableAllEntities,
			YRotation: mc.Optional[mc.FloatRange]{
				Present: true,
				Value: mc.FloatRange{
					Min: mc.Optional[float64]{Value: -10, Present: true},
					Max: mc.Optional[float64]{Value: 10, Present: true},
				},
			},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != south {
			t.Fatalf("want only south-facing, got %v", got)
		}
	})

	t.Run("x_rotation_filters_by_pitch", func(t *testing.T) {
		w := NewWorld()
		level := makeEntityWithRot(entity.ZombieID, [3]float64{0, 0, 0}, 0, 0)
		looking_down := makeEntityWithRot(entity.ZombieID, [3]float64{1, 0, 0}, 0, 80)
		addEntity(w, level)
		addEntity(w, looking_down)

		sel := &mc.Selector{
			Variable: mc.SelectorVariableAllEntities,
			XRotation: mc.Optional[mc.FloatRange]{
				Present: true,
				Value: mc.FloatRange{
					Min: mc.Optional[float64]{Value: 45, Present: true},
				},
			},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != looking_down {
			t.Fatalf("want only looking-down, got %v", got)
		}
	})
}

func TestResolveSelectorLevel(t *testing.T) {
	t.Run("range_includes_player", func(t *testing.T) {
		w := NewWorld()
		low := makePlayer("low", [3]float64{0, 0, 0}, 3, 0)
		high := makePlayer("high", [3]float64{1, 0, 0}, 30, 0)
		addEntity(w, low)
		addEntity(w, high)

		sel := &mc.Selector{
			Variable: mc.SelectorVariableAllPlayers,
			Level: mc.Optional[mc.IntRange]{
				Present: true,
				Value: mc.IntRange{
					Min: mc.Optional[int]{Value: 10, Present: true},
				},
			},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != high {
			t.Fatalf("want only high-level, got %v", got)
		}
	})

	t.Run("filter_rejects_non_player", func(t *testing.T) {
		// @e[level=..5] over a zombie — level filter must reject non-players.
		w := NewWorld()
		zombie := makeEntity(entity.ZombieID, [3]float64{0, 0, 0})
		addEntity(w, zombie)

		sel := &mc.Selector{
			Variable: mc.SelectorVariableAllEntities,
			Level: mc.Optional[mc.IntRange]{
				Present: true,
				Value: mc.IntRange{
					Max: mc.Optional[int]{Value: 5, Present: true},
				},
			},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 0 {
			t.Fatalf("want empty, got %v", got)
		}
	})
}

func TestResolveSelectorGamemode(t *testing.T) {
	t.Run("matches_named_gamemode", func(t *testing.T) {
		w := NewWorld()
		surv := makePlayer("surv", [3]float64{0, 0, 0}, 0, 0)
		crea := makePlayer("crea", [3]float64{1, 0, 0}, 0, 1)
		spec := makePlayer("spec", [3]float64{2, 0, 0}, 0, 3)
		addEntity(w, surv)
		addEntity(w, crea)
		addEntity(w, spec)

		sel := &mc.Selector{
			Variable: mc.SelectorVariableAllPlayers,
			Gamemode: mc.Optional[string]{Value: "creative", Present: true},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 1 || got[0] != crea {
			t.Fatalf("want only creative player, got %v", got)
		}
	})

	t.Run("filter_rejects_non_player", func(t *testing.T) {
		w := NewWorld()
		zombie := makeEntity(entity.ZombieID, [3]float64{0, 0, 0})
		addEntity(w, zombie)

		sel := &mc.Selector{
			Variable: mc.SelectorVariableAllEntities,
			Gamemode: mc.Optional[string]{Value: "survival", Present: true},
		}
		got := w.resolveSelector(sel, uuid.Nil, [3]float64{})
		if len(got) != 0 {
			t.Fatalf("want empty (zombie has no gamemode), got %v", got)
		}
	})
}
