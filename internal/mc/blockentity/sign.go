package blockentity

import (
	"io"

	"github.com/Tnze/go-mc/nbt"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

const (
	signLines        = 4
	signDefaultColor = "black"
)

type signText struct {
	messages [signLines]tc.Component
	color    string
	glowing  bool
}

type Sign struct {
	pos         world.BlockPos
	front, back signText
	waxed       bool
	editorID    int32 // entity ID of the player allowed to edit
	hasEditor   bool
}

func NewSign(pos world.BlockPos) *Sign {
	s := &Sign{pos: pos}
	s.front.color = signDefaultColor
	s.back.color = signDefaultColor
	return s
}

func (s *Sign) Pos() world.BlockPos { return s.pos }

func (s *Sign) Type() world.BEType { return TypeSign }

func (s *Sign) SetEditor(entityID int32) {
	s.editorID = entityID
	s.hasEditor = true
}

func (s *Sign) OnInteract(ctx world.InteractContext) world.PlaceResult {
	if s.waxed || ctx.Player == nil {
		return world.PlaceResult{OK: true}
	}
	s.SetEditor(ctx.Player.EntityID)
	pos := s.pos
	return world.PlaceResult{OK: true, OpenSignEdit: &pos}
}

// ApplyEdit updates one side's text from a ServerboundUpdateSign. It returns false
// (rejecting the edit) when the sign is waxed, no editor is armed, or the sender is
// not the armed editor. A successful edit clears the editor.
func (s *Sign) ApplyEdit(senderID int32, front bool, lines [signLines]string) bool {
	if s.waxed || !s.hasEditor || s.editorID != senderID {
		return false
	}
	side := &s.front
	if !front {
		side = &s.back
	}
	for i := 0; i < signLines; i++ {
		side.messages[i] = tc.Text(lines[i])
	}
	s.hasEditor = false
	return true
}

// TODO: encode full text-component NBT per line once a list-of-component encoder lands.
type sideNetworkData struct {
	Messages       []string `nbt:"messages"`
	Color          string   `nbt:"color"`
	HasGlowingText bool     `nbt:"has_glowing_text"`
}

type signNetworkData struct {
	FrontText sideNetworkData `nbt:"front_text"`
	BackText  sideNetworkData `nbt:"back_text"`
	IsWaxed   bool            `nbt:"is_waxed"`
}

func (d signNetworkData) WriteTo(w io.Writer) (int64, error) {
	enc := nbt.NewEncoder(w)
	enc.NetworkFormat(true)
	if err := enc.Encode(d, ""); err != nil {
		return 0, err
	}
	return 0, nil
}

func (s *signText) networkData() sideNetworkData {
	msgs := make([]string, signLines)
	for i := range s.messages {
		if s.messages[i] != nil {
			msgs[i] = s.messages[i].String()
		}
	}
	return sideNetworkData{Messages: msgs, Color: s.color, HasGlowingText: s.glowing}
}

func (s *Sign) NetworkData() any {
	return signNetworkData{
		FrontText: s.front.networkData(),
		BackText:  s.back.networkData(),
		IsWaxed:   s.waxed,
	}
}

func newSignBE(pos world.BlockPos, _ any) (world.BlockEntity, error) {
	return NewSign(pos), nil
}
