package text_component

import "fmt"

func PresetPlayerName(name string) *TextComponent {
	return Text(name).SuggestCommand(fmt.Sprintf("/tell %s ", name))
}

// Container creates a empty text component to wrap other components. Useful for styling
func Container(children ...Component) *TextComponent {
	return Text("").AddExtra(children...)
}
