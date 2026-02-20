package text_component

import "fmt"

func PresetPlayerName(name string) *TextComponent {
	return Text(name).SuggestCommand(fmt.Sprintf("/tell %s ", name))
}
