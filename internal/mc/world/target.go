package world

import (
	"math"
	"math/rand/v2"
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	"github.com/Gagonlaire/mcgoserv/internal/mc/nbtpath"
	"github.com/google/uuid"
)

// ResolveTarget resolves a target to all matching entities. Player-only selectors
// (@p, @r, @a) -> player index, entity selectors (@e, @n) -> full entity index. @s -> source
// UUID -> full entity index, name targets only match players
func (w *World) ResolveTarget(target *mc.EntityTarget, sourceUUID uuid.UUID, sourcePos [3]float64) []entity.Entity {
	switch target.Type {
	case mc.TargetTypePlayerName:
		for _, p := range w.PlayersByID {
			if p.Name == target.Name {
				return []entity.Entity{p}
			}
		}
		return nil
	case mc.TargetTypeUUID:
		if e := w.EntitiesByUUID[target.UUID]; e != nil {
			return []entity.Entity{e}
		}
		return nil
	case mc.TargetTypeSelector:
		return w.resolveSelector(target.Selector, sourceUUID, sourcePos)
	}
	return nil
}

func (w *World) ResolvePlayers(target *mc.EntityTarget, sourceUUID uuid.UUID, sourcePos [3]float64) []*entity.Player {
	ents := w.ResolveTarget(target, sourceUUID, sourcePos)
	if len(ents) == 0 {
		return nil
	}
	players := make([]*entity.Player, 0, len(ents))
	for _, e := range ents {
		if p, ok := e.(*entity.Player); ok {
			players = append(players, p)
		}
	}
	return players
}

func (w *World) resolveSelector(sel *mc.Selector, sourceUUID uuid.UUID, sourcePos [3]float64) []entity.Entity {
	refPos := selectorReferencePoint(sel, sourcePos)

	if sel.Variable == mc.SelectorVariableSelf {
		e := w.EntitiesByUUID[sourceUUID]
		if e == nil || !passesFilter(e, sel, refPos) {
			return nil
		}
		return []entity.Entity{e}
	}

	candidates := w.selectorCandidates(sel)
	filtered := candidates[:0]
	for _, e := range candidates {
		if passesFilter(e, sel, refPos) {
			filtered = append(filtered, e)
		}
	}
	return reduceSelectorResult(sel, filtered, refPos)
}

// selectorReferencePoint resolves the reference point used by distance and volume filters
func selectorReferencePoint(sel *mc.Selector, sourcePos [3]float64) [3]float64 {
	ref := sourcePos
	if sel.X.Present {
		ref[0] = sel.X.Value
	}
	if sel.Y.Present {
		ref[1] = sel.Y.Value
	}
	if sel.Z.Present {
		ref[2] = sel.Z.Value
	}
	return ref
}

func (w *World) selectorCandidates(sel *mc.Selector) []entity.Entity {
	switch sel.Variable {
	case mc.SelectorVariableAllPlayers,
		mc.SelectorVariableNearestPlayer,
		mc.SelectorVariableRandomPlayer:
		out := make([]entity.Entity, 0, len(w.PlayersByID))
		for _, p := range w.PlayersByID {
			out = append(out, p)
		}
		return out
	case mc.SelectorVariableAllEntities,
		mc.SelectorVariableNearestEntity:
		out := make([]entity.Entity, 0, len(w.EntitiesByID))
		for _, e := range w.EntitiesByID {
			out = append(out, e)
		}
		return out
	}
	return nil
}

func reduceSelectorResult(sel *mc.Selector, filtered []entity.Entity, sourcePos [3]float64) []entity.Entity {
	switch sel.Variable {
	case mc.SelectorVariableAllPlayers, mc.SelectorVariableAllEntities:
		return filtered
	case mc.SelectorVariableNearestPlayer, mc.SelectorVariableNearestEntity:
		return nearestOf(filtered, sourcePos)
	case mc.SelectorVariableRandomPlayer:
		if len(filtered) == 0 {
			return nil
		}
		return []entity.Entity{filtered[rand.IntN(len(filtered))]}
	}
	return nil
}

func nearestOf(ents []entity.Entity, pos [3]float64) []entity.Entity {
	var nearest entity.Entity
	bestDist := math.MaxFloat64
	for _, e := range ents {
		d := distSq(pos, e.Base().Position)
		if d < bestDist {
			bestDist = d
			nearest = e
		}
	}
	if nearest == nil {
		return nil
	}
	return []entity.Entity{nearest}
}

// passesFilter reports whether entity e satisfies every filter argument in sel.
// Filter args are applied independently of the selector variable; the variable
// only governs candidate enumeration and reduction.
func passesFilter(e entity.Entity, sel *mc.Selector, refPos [3]float64) bool {
	if !passesTypeFilter(e, sel) {
		return false
	}
	if !passesNbtFilter(e, sel) {
		return false
	}
	if !passesDistanceFilter(e, sel, refPos) {
		return false
	}
	if !passesVolumeFilter(e, sel, refPos) {
		return false
	}
	if !passesRotationFilter(e, sel) {
		return false
	}
	if !passesLevelFilter(e, sel) {
		return false
	}
	if !passesGamemodeFilter(e, sel) {
		return false
	}
	return true
}

func passesLevelFilter(e entity.Entity, sel *mc.Selector) bool {
	if !sel.Level.Present {
		return true
	}
	p, ok := e.(*entity.Player)
	if !ok {
		return false
	}
	v := int(p.XpLevel)
	r := sel.Level.Value
	if r.Min.Present && v < r.Min.Value {
		return false
	}
	if r.Max.Present && v > r.Max.Value {
		return false
	}
	return true
}

func passesGamemodeFilter(e entity.Entity, sel *mc.Selector) bool {
	if !sel.Gamemode.Present {
		return true
	}
	p, ok := e.(*entity.Player)
	if !ok {
		return false
	}
	want, ok := gamemodeID(sel.Gamemode.Value)
	if !ok {
		return false
	}
	return p.GameMode == want
}

func gamemodeID(name string) (int32, bool) {
	switch name {
	case "survival":
		return 0, true
	case "creative":
		return 1, true
	case "adventure":
		return 2, true
	case "spectator":
		return 3, true
	}
	return 0, false
}

func passesDistanceFilter(e entity.Entity, sel *mc.Selector, refPos [3]float64) bool {
	if !sel.Distance.Present {
		return true
	}
	d := math.Sqrt(distSq(refPos, e.GetPos()))
	r := sel.Distance.Value
	if r.Min.Present && d < r.Min.Value {
		return false
	}
	if r.Max.Present && d > r.Max.Value {
		return false
	}
	return true
}

func passesVolumeFilter(e entity.Entity, sel *mc.Selector, refPos [3]float64) bool {
	if !sel.Dx.Present && !sel.Dy.Present && !sel.Dz.Present {
		return true
	}
	dx, dy, dz := 0.0, 0.0, 0.0
	if sel.Dx.Present {
		dx = sel.Dx.Value
	}
	if sel.Dy.Present {
		dy = sel.Dy.Value
	}
	if sel.Dz.Present {
		dz = sel.Dz.Value
	}
	lo, hi := normalizeBox(refPos, [3]float64{dx, dy, dz})

	pos := e.GetPos()
	width := e.GetType().Width()
	height := e.GetType().Height()
	half := width / 2
	emin := [3]float64{pos[0] - half, pos[1], pos[2] - half}
	emax := [3]float64{pos[0] + half, pos[1] + height, pos[2] + half}

	for i := 0; i < 3; i++ {
		if emax[i] < lo[i] || emin[i] > hi[i] {
			return false
		}
	}
	return true
}

func normalizeBox(corner, size [3]float64) (lo, hi [3]float64) {
	for i := 0; i < 3; i++ {
		a, b := corner[i], corner[i]+size[i]
		if a <= b {
			lo[i], hi[i] = a, b
		} else {
			lo[i], hi[i] = b, a
		}
	}
	return lo, hi
}

func passesRotationFilter(e entity.Entity, sel *mc.Selector) bool {
	rot := e.GetRot()
	yaw := float64(rot[0])
	pitch := float64(rot[1])
	if sel.XRotation.Present && !inFloatRange(pitch, sel.XRotation.Value) {
		return false
	}
	if sel.YRotation.Present && !inFloatRange(yaw, sel.YRotation.Value) {
		return false
	}
	return true
}

func inFloatRange(v float64, r mc.FloatRange) bool {
	if r.Min.Present && v < r.Min.Value {
		return false
	}
	if r.Max.Present && v > r.Max.Value {
		return false
	}
	return true
}

func passesNbtFilter(e entity.Entity, sel *mc.Selector) bool {
	if len(sel.NbtIncludes) == 0 && len(sel.NbtExcludes) == 0 {
		return true
	}
	src, ok := e.(nbtpath.NbtSource)
	if !ok {
		return false
	}
	root, err := src.NbtRoot()
	if err != nil {
		return false
	}
	for _, inc := range sel.NbtIncludes {
		if !nbtpath.Match(root, inc) {
			return false
		}
	}
	for _, exc := range sel.NbtExcludes {
		if nbtpath.Match(root, exc) {
			return false
		}
	}
	return true
}

func passesTypeFilter(e entity.Entity, sel *mc.Selector) bool {
	if !sel.TypeInclude.Present && len(sel.TypeExclude) == 0 {
		return true
	}
	name := e.GetType().Name()
	if sel.TypeInclude.Present && sel.TypeInclude.Value != name {
		return false
	}
	for _, ex := range sel.TypeExclude {
		if ex == name {
			return false
		}
	}
	return true
}

func (w *World) ResolveMessage(msg *mc.ParsedMessage, sourceUUID uuid.UUID, sourcePos [3]float64) string {
	if len(msg.Selectors) == 0 {
		return msg.Raw
	}
	var b strings.Builder
	b.Grow(len(msg.Raw))
	prev := 0
	for _, span := range msg.Selectors {
		b.WriteString(msg.Raw[prev:span.Start])
		ents := w.resolveSelector(span.Selector, sourceUUID, sourcePos)
		first := true
		for _, e := range ents {
			p, ok := e.(*entity.Player)
			if !ok {
				continue
			}
			if !first {
				b.WriteString(", ")
			}
			b.WriteString(p.Name)
			first = false
		}
		prev = span.End
	}
	b.WriteString(msg.Raw[prev:])
	return b.String()
}

func distSq(a, b [3]float64) float64 {
	dx := a[0] - b[0]
	dy := a[1] - b[1]
	dz := a[2] - b[2]
	return dx*dx + dy*dy + dz*dz
}
