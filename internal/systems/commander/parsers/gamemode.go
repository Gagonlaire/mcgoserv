package parsers

import (
	"io"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

type GamemodeType struct{}

var Gamemode = GamemodeType{}

var gamemodeByName = map[string]int32{
	"survival":  0,
	"creative":  1,
	"adventure": 2,
	"spectator": 3,
}

func (GamemodeType) ID() int { return 42 } // minecraft:gamemode

func (GamemodeType) Parse(r *commander.CommandReader) (any, error) {
	start := r.Cursor()
	raw := r.ReadUnquotedString()
	if mode, ok := gamemodeByName[raw]; ok {
		return mode, nil
	}
	r.SetCursor(start)
	return nil, commander.NewParsingErrorAt(
		r, tc.Translatable(mcdata.ArgumentGamemodeInvalid, tc.Text(raw)), start,
	)
}

func (GamemodeType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }
