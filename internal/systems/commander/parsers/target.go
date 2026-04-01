package parsers

import (
	"io"
	"strconv"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/google/uuid"
)

type EntityType struct {
	single      bool
	playersOnly bool
}

type GameProfileType struct{}

type UUIDType struct{}

const (
	EntityTargetFlagSingle      = 0x01
	EntityTargetFlagPlayersOnly = 0x02
)

var Entity = EntityType{}

var UUID = UUIDType{}

var GameProfile = GameProfileType{}

func (e EntityType) ID() int { return 6 } // minecraft:entity

func (e EntityType) Parse(r *commander.CommandReader) (any, error) {
	if !r.CanRead() {
		return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ArgumentEntityInvalid), r.Input(), r.Cursor())
	}

	start := r.Cursor()

	if r.Peek() == '@' {
		sel, err := parseSelector(r)
		if err != nil {
			return nil, err
		}

		if e.playersOnly {
			v := sel.Variable
			if v == mc.SelectorVariableAllEntities || v == mc.SelectorVariableNearestEntity {
				r.SetCursor(start)
				return nil, commander.NewParsingErrorAt(
					tc.Translatable(mcdata.ArgumentPlayerEntities),
					r.Input(), start,
				)
			}
		}

		if e.single {
			v := sel.Variable
			multiTarget := v == mc.SelectorVariableAllEntities || v == mc.SelectorVariableAllPlayers
			if multiTarget && !sel.Limit.Present {
				r.SetCursor(start)
				key := mcdata.ArgumentEntityToomany
				if e.playersOnly {
					key = mcdata.ArgumentPlayerToomany
				}
				return nil, commander.NewParsingErrorAt(
					tc.Translatable(key),
					r.Input(), start,
				)
			}
			if sel.Limit.Present && sel.Limit.Value != 1 {
				r.SetCursor(start)
				key := mcdata.ArgumentEntityToomany
				if e.playersOnly {
					key = mcdata.ArgumentPlayerToomany
				}
				return nil, commander.NewParsingErrorAt(
					tc.Translatable(key),
					r.Input(), start,
				)
			}
		}

		return &mc.EntityTarget{
			Type:     mc.TargetTypeSelector,
			Selector: sel,
		}, nil
	}

	if e.playersOnly && isUUIDCandidate(r) {
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentPlayerUnknown),
			r.Input(), start,
		)
	}

	if isUUIDCandidate(r) {
		uuidStr := readUUID(r)
		if uuidStr != "" {
			id, err := uuid.Parse(uuidStr)
			if err != nil {
				return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ArgumentUuidInvalid), r.Input(), start)
			}
			return &mc.EntityTarget{
				Type: mc.TargetTypeUUID,
				UUID: id,
			}, nil
		}
		r.SetCursor(start)
	}

	name := r.ReadUnquotedString()
	if len(name) == 0 {
		return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ArgumentEntityInvalid), r.Input(), start)
	}

	return &mc.EntityTarget{
		Type: mc.TargetTypePlayerName,
		Name: name,
	}, nil
}

func (e EntityType) WriteTo(w io.Writer) (int64, error) {
	var flags byte
	if e.single {
		flags |= EntityTargetFlagSingle
	}
	if e.playersOnly {
		flags |= EntityTargetFlagPlayersOnly
	}
	return mc.Byte(flags).WriteTo(w)
}

func (e EntityType) Single(v bool) EntityType {
	e.single = v
	return e
}

func (e EntityType) PlayersOnly(v bool) EntityType {
	e.playersOnly = v
	return e
}

func (u UUIDType) ID() int { return 56 } // minecraft:uuid

func (u UUIDType) Parse(r *commander.CommandReader) (any, error) {
	start := r.Cursor()

	if !isUUIDCandidate(r) {
		return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ArgumentUuidInvalid), r.Input(), start)
	}

	uuidStr := readUUID(r)
	if uuidStr == "" {
		return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ArgumentUuidInvalid), r.Input(), start)
	}

	id, err := uuid.Parse(uuidStr)
	if err != nil {
		return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ArgumentUuidInvalid), r.Input(), start)
	}

	return id, nil
}

func (u UUIDType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }

func (g GameProfileType) ID() int { return 7 } // minecraft:game_profile

func (g GameProfileType) Parse(r *commander.CommandReader) (any, error) {
	if !r.CanRead() {
		return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ArgumentEntityInvalid), r.Input(), r.Cursor())
	}

	start := r.Cursor()

	if r.Peek() == '@' {
		sel, err := parseSelector(r)
		if err != nil {
			return nil, err
		}

		if sel.Variable == mc.SelectorVariableAllEntities || sel.Variable == mc.SelectorVariableNearestEntity {
			r.SetCursor(start)
			return nil, commander.NewParsingError(
				tc.Translatable(mcdata.ArgumentPlayerEntities),
				r.Input(),
			)
		}

		return &mc.EntityTarget{
			Type:     mc.TargetTypeSelector,
			Selector: sel,
		}, nil
	}

	if isUUIDCandidate(r) {
		uuidStr := readUUID(r)
		if uuidStr != "" {
			id, err := uuid.Parse(uuidStr)
			if err != nil {
				return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ArgumentUuidInvalid), r.Input(), start)
			}
			return &mc.EntityTarget{
				Type: mc.TargetTypeUUID,
				UUID: id,
			}, nil
		}
		r.SetCursor(start)
	}

	name := r.ReadUnquotedString()
	if len(name) == 0 {
		return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ArgumentEntityInvalid), r.Input(), start)
	}

	return &mc.EntityTarget{
		Type: mc.TargetTypePlayerName,
		Name: name,
	}, nil
}

func (g GameProfileType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }

func parseSelector(r *commander.CommandReader) (*mc.Selector, error) {
	if !r.CanRead() || r.Peek() != '@' {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentEntitySelectorMissing),
			r.Input(), r.Cursor(),
		)
	}
	r.Skip()

	if !r.CanRead() {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentEntitySelectorMissing),
			r.Input(), r.Cursor(),
		)
	}

	varByte := r.Read()
	if !mc.ValidSelectorVariable(varByte) {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentEntitySelectorUnknown, tc.Text(string(varByte))),
			r.Input(), r.Cursor()-1,
		)
	}

	sel := &mc.Selector{
		Variable: mc.SelectorVariable(varByte),
	}

	if r.CanRead() && r.Peek() == '[' {
		r.Skip()
		if err := parseSelectorOptions(r, sel); err != nil {
			return nil, err
		}
	}

	return sel, nil
}

func parseSelectorOptions(r *commander.CommandReader, sel *mc.Selector) error {
	for {
		r.SkipWhitespace()

		if !r.CanRead() {
			return commander.NewParsingErrorAt(
				tc.Translatable(mcdata.ArgumentEntityOptionsUnterminated),
				r.Input(), r.Cursor(),
			)
		}

		if r.Peek() == ']' {
			r.Skip()
			return nil
		}

		keyStart := r.Cursor()
		key := readOptionKey(r)
		if len(key) == 0 {
			return commander.NewParsingErrorAt(
				tc.Translatable(mcdata.ArgumentEntityOptionsUnknown, tc.Text("")),
				r.Input(), keyStart,
			)
		}

		r.SkipWhitespace()
		if !r.CanRead() || r.Peek() != '=' {
			return commander.NewParsingErrorAt(
				tc.Translatable(mcdata.ArgumentEntityOptionsValueless, tc.Text(key)),
				r.Input(), r.Cursor(),
			)
		}
		r.Skip()
		r.SkipWhitespace()

		if err := parseSelectorOption(r, sel, key, keyStart); err != nil {
			return err
		}

		r.SkipWhitespace()
		if r.CanRead() && r.Peek() == ',' {
			r.Skip()
		}
	}
}

func parseSelectorOption(r *commander.CommandReader, sel *mc.Selector, key string, keyStart int) error {
	inapplicable := func(present bool) error {
		if present {
			return commander.NewParsingErrorAt(
				tc.Translatable(mcdata.ArgumentEntityOptionsInapplicable, tc.Text(key)),
				r.Input(), keyStart,
			)
		}
		return nil
	}

	switch key {
	case "x":
		if err := inapplicable(sel.X.Present); err != nil {
			return err
		}
		return parseSelectorFloat64(r, &sel.X)
	case "y":
		if err := inapplicable(sel.Y.Present); err != nil {
			return err
		}
		return parseSelectorFloat64(r, &sel.Y)
	case "z":
		if err := inapplicable(sel.Z.Present); err != nil {
			return err
		}
		return parseSelectorFloat64(r, &sel.Z)
	case "dx":
		if err := inapplicable(sel.Dx.Present); err != nil {
			return err
		}
		return parseSelectorFloat64(r, &sel.Dx)
	case "dy":
		if err := inapplicable(sel.Dy.Present); err != nil {
			return err
		}
		return parseSelectorFloat64(r, &sel.Dy)
	case "dz":
		if err := inapplicable(sel.Dz.Present); err != nil {
			return err
		}
		return parseSelectorFloat64(r, &sel.Dz)
	case "distance":
		if err := inapplicable(sel.Distance.Present); err != nil {
			return err
		}
		return parseSelectorRange(r, &sel.Distance, true)
	case "x_rotation":
		if err := inapplicable(sel.XRotation.Present); err != nil {
			return err
		}
		return parseSelectorRange(r, &sel.XRotation, false)
	case "y_rotation":
		if err := inapplicable(sel.YRotation.Present); err != nil {
			return err
		}
		return parseSelectorRange(r, &sel.YRotation, false)
	case "level":
		if err := inapplicable(sel.Level.Present); err != nil {
			return err
		}
		return parseSelectorIntRange(r, &sel.Level)
	case "limit":
		if err := inapplicable(sel.Limit.Present); err != nil {
			return err
		}
		return parseSelectorInt(r, &sel.Limit, key)
	case "sort":
		if err := inapplicable(sel.Sort.Present); err != nil {
			return err
		}
		return parseSelectorSort(r, sel)
	case "gamemode":
		if err := inapplicable(sel.Gamemode.Present); err != nil {
			return err
		}
		return parseSelectorGamemode(r, sel)
	default:
		return commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentEntityOptionsUnknown, tc.Text(key)),
			r.Input(), keyStart,
		)
	}
}

func parseSelectorFloat64(r *commander.CommandReader, target *mc.Optional[float64]) error {
	start := r.Cursor()
	raw := readOptionValue(r)
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ParsingDoubleInvalid, tc.Text(raw)),
			r.Input(), start,
		)
	}
	target.Value = val
	target.Present = true
	return nil
}

func parseSelectorInt(r *commander.CommandReader, target *mc.Optional[int], key string) error {
	start := r.Cursor()
	raw := readOptionValue(r)
	val, err := strconv.Atoi(raw)
	if err != nil {
		return commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ParsingIntInvalid, tc.Text(raw)),
			r.Input(), start,
		)
	}
	if key == "limit" && val < 1 {
		return commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentEntityOptionsLimitToosmall),
			r.Input(), start,
		)
	}
	target.Value = val
	target.Present = true
	return nil
}

func parseSelectorSort(r *commander.CommandReader, sel *mc.Selector) error {
	start := r.Cursor()
	raw := readOptionValue(r)
	switch raw {
	case "nearest", "furthest", "random", "arbitrary":
		sel.Sort = mc.Optional[string]{Value: raw, Present: true}
		return nil
	default:
		return commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentEntityOptionsSortIrreversible, tc.Text(raw)),
			r.Input(), start,
		)
	}
}

func parseSelectorGamemode(r *commander.CommandReader, sel *mc.Selector) error {
	start := r.Cursor()
	raw := readOptionValue(r)
	switch raw {
	case "survival", "creative", "adventure", "spectator":
		sel.Gamemode = mc.Optional[string]{Value: raw, Present: true}
		return nil
	default:
		return commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentEntityOptionsModeInvalid, tc.Text(raw)),
			r.Input(), start,
		)
	}
}

func parseSelectorRange(r *commander.CommandReader, target *mc.Optional[mc.FloatRange], nonNegative bool) error {
	start := r.Cursor()
	raw := readOptionValue(r)

	lo, hi, err := parseRange(raw, func(s string) (float64, error) {
		return strconv.ParseFloat(s, 64)
	})
	nr := mc.FloatRange{Min: lo, Max: hi}
	if err != nil {
		return commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ParsingDoubleInvalid, tc.Text(raw)),
			r.Input(), start,
		)
	}

	if nonNegative && nr.Min.Present && nr.Min.Value < 0 {
		return commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentEntityOptionsDistanceNegative),
			r.Input(), start,
		)
	}

	target.Value = nr
	target.Present = true
	return nil
}

func parseSelectorIntRange(r *commander.CommandReader, target *mc.Optional[mc.IntRange]) error {
	start := r.Cursor()
	raw := readOptionValue(r)

	lo, hi, err := parseRange(raw, strconv.Atoi)
	nr := mc.IntRange{Min: lo, Max: hi}
	if err != nil {
		return commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ParsingIntInvalid, tc.Text(raw)),
			r.Input(), start,
		)
	}

	if nr.Min.Present && nr.Min.Value < 0 {
		return commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentEntityOptionsLevelNegative),
			r.Input(), start,
		)
	}

	target.Value = nr
	target.Present = true
	return nil
}

func readOptionKey(r *commander.CommandReader) string {
	start := r.Cursor()
	for r.CanRead() {
		ch := r.Peek()
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_' {
			r.Skip()
		} else {
			break
		}
	}
	return r.Input()[start:r.Cursor()]
}

func readOptionValue(r *commander.CommandReader) string {
	if !r.CanRead() {
		return ""
	}

	if r.Peek() == '"' || r.Peek() == '\'' {
		val, err := r.ReadQuotedString()
		if err != nil {
			return ""
		}
		return val
	}

	start := r.Cursor()
	depth := 0
	for r.CanRead() {
		ch := r.Peek()
		if ch == '{' || ch == '[' {
			depth++
			r.Skip()
		} else if ch == '}' || ch == ']' {
			if depth == 0 {
				break
			}
			depth--
			r.Skip()
		} else if ch == ',' && depth == 0 {
			break
		} else {
			r.Skip()
		}
	}
	return r.Input()[start:r.Cursor()]
}

func isUUIDCandidate(r *commander.CommandReader) bool {
	if r.RemainingLength() < 36 {
		return false
	}
	remaining := r.GetRemaining()
	return len(remaining) >= 36 && remaining[8] == '-' && remaining[13] == '-' &&
		remaining[18] == '-' && remaining[23] == '-'
}

func readUUID(r *commander.CommandReader) string {
	start := r.Cursor()
	remaining := r.GetRemaining()

	if len(remaining) < 36 {
		return ""
	}

	candidate := remaining[:36]
	for i, ch := range candidate {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if ch != '-' {
				r.SetCursor(start)
				return ""
			}
		} else if !isHexDigit(byte(ch)) {
			r.SetCursor(start)
			return ""
		}
	}

	r.SetCursor(start + 36)

	if r.CanRead() && commander.IsAllowedInUnquotedString(r.Peek()) {
		r.SetCursor(start)
		return ""
	}

	return candidate
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
