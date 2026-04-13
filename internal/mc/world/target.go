package world

import (
	"math"
	"math/rand/v2"
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	"github.com/google/uuid"
)

// ResolveTarget todo: it should take a generic entity as the source
func (w *World) ResolveTarget(target *mc.EntityTarget, sourceUUID uuid.UUID, sourcePos [3]float64) []*entities.Player {
	switch target.Type {
	case mc.TargetTypePlayerName:
		for _, p := range w.PlayersByID {
			if p.Name == target.Name {
				return []*entities.Player{p}
			}
		}
		return nil
	case mc.TargetTypeUUID:
		if p := w.PlayersByUUID[target.UUID]; p != nil {
			return []*entities.Player{p}
		}
		return nil
	case mc.TargetTypeSelector:
		return w.resolveSelector(target.Selector, sourceUUID, sourcePos)
	}

	return nil
}

// todo: this should return a list of entities, not just players
func (w *World) resolveSelector(sel *mc.Selector, sourceUUID uuid.UUID, sourcePos [3]float64) []*entities.Player {
	switch sel.Variable {
	case mc.SelectorVariableSelf:
		if p := w.PlayersByUUID[sourceUUID]; p != nil {
			return []*entities.Player{p}
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
		return []*entities.Player{players[rand.IntN(len(players))]}
	}

	return nil
}

func (w *World) nearestPlayer(pos [3]float64) []*entities.Player {
	var nearest *entities.Player
	bestDist := math.MaxFloat64
	for _, p := range w.PlayersByID {
		d := distSq(pos, p.Position)
		if d < bestDist {
			bestDist = d
			nearest = p
		}
	}
	if nearest == nil {
		return nil
	}
	return []*entities.Player{nearest}
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
		players := w.resolveSelector(span.Selector, sourceUUID, sourcePos)
		if len(players) == 1 {
			b.WriteString(players[0].Name)
		} else {
			for i, p := range players {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(p.Name)
			}
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
