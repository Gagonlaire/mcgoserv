package world

import (
	"math"
	"math/rand/v2"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/google/uuid"
)

// ResolveTarget todo: it should take a generic entity as the source
func (w *World) ResolveTarget(target *mc.EntityTarget, sourceUUID uuid.UUID, sourcePos [3]float64) []*Player {
	switch target.Type {
	case mc.TargetTypePlayerName:
		for _, p := range w.PlayersByID {
			if p.Name == target.Name {
				return []*Player{p}
			}
		}
		return nil
	case mc.TargetTypeUUID:
		if p := w.PlayersByUUID[target.UUID]; p != nil {
			return []*Player{p}
		}
		return nil
	case mc.TargetTypeSelector:
		return w.resolveSelector(target.Selector, sourceUUID, sourcePos)
	}

	return nil
}

func (w *World) resolveSelector(sel *mc.Selector, sourceUUID uuid.UUID, sourcePos [3]float64) []*Player {
	switch sel.Variable {
	case mc.SelectorVariableSelf:
		if p := w.PlayersByUUID[sourceUUID]; p != nil {
			return []*Player{p}
		}
		return nil
	case mc.SelectorVariableAllPlayers, mc.SelectorVariableAllEntities:
		return w.Players()
	case mc.SelectorVariableNearestPlayer, mc.SelectorVariableNearestEntity:
		return w.nearestPlayer(sourcePos)
	case mc.SelectorVariableRandomPlayer:
		players := w.Players()
		if len(players) == 0 {
			return nil
		}
		return []*Player{players[rand.IntN(len(players))]}
	}

	return nil
}

func (w *World) nearestPlayer(pos [3]float64) []*Player {
	var nearest *Player
	bestDist := math.MaxFloat64
	for _, p := range w.PlayersByID {
		d := distSq(pos, p.Pos)
		if d < bestDist {
			bestDist = d
			nearest = p
		}
	}
	if nearest == nil {
		return nil
	}
	return []*Player{nearest}
}

func distSq(a, b [3]float64) float64 {
	dx := a[0] - b[0]
	dy := a[1] - b[1]
	dz := a[2] - b[2]
	return dx*dx + dy*dy + dz*dz
}
