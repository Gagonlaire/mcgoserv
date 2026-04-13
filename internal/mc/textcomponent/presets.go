package textcomponent

import "fmt"

func PlayerName(name string) *TextComponent {
	return Text(name).SuggestCommand(fmt.Sprintf("/tell %s ", name)).SetInsertion(name)
}

func Space(count int) *TextComponent {
	return Text(fmt.Sprintf("%*s", count, ""))
}

// Container creates a empty text component to wrap other components. Useful for styling
func Container(children ...Component) *TextComponent {
	return Text("").AddExtra(children...)
}
