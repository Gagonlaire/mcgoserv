package textcomponent

type Interactivity struct {
	ClickEvent any    `nbt:"click_event,omitempty" json:"click_event,omitempty"`
	HoverEvent any    `nbt:"hover_event,omitempty" json:"hover_event,omitempty"`
	Insertion  string `nbt:"insertion,omitempty" json:"insertion,omitempty"`
}

// todo: implement missing event
type Event struct {
	Action string `nbt:"action" json:"action"`
}

type ClickOpenURL struct {
	Event
	Url string `nbt:"url" json:"url"`
}

type ClickCopyToClipboard struct {
	Event
	Value string `nbt:"value" json:"value"`
}

type ClickSuggestOrRunCommand struct {
	Event
	Command string `nbt:"command" json:"command"`
}

type HoverEventShowText struct {
	Value Component `nbt:"value" json:"value"`
	Event
}

type HoverEventShowItem struct {
	Event
	Id    string `nbt:"id" json:"id"`
	Count int32  `nbt:"count" json:"count"`
	// todo: implement data component
}

func (b *Base[T]) OpenURL(url string) T {
	b.ClickEvent = ClickOpenURL{
		Event: Event{
			Action: "open_url",
		},
		Url: url,
	}
	return b.self
}

func (b *Base[T]) CopyToClipboard(value string) T {
	b.ClickEvent = ClickCopyToClipboard{
		Event: Event{
			Action: "copy_to_clipboard",
		},
		Value: value,
	}
	return b.self
}

func (b *Base[T]) RunCommand(command string) T {
	b.ClickEvent = ClickSuggestOrRunCommand{
		Event: Event{
			Action: "run_command",
		},
		Command: command,
	}
	return b.self
}

func (b *Base[T]) SuggestCommand(command string) T {
	b.ClickEvent = ClickSuggestOrRunCommand{
		Event: Event{
			Action: "suggest_command",
		},
		Command: command,
	}
	return b.self
}

func (b *Base[T]) ShowText(text Component) T {
	b.HoverEvent = HoverEventShowText{
		Event: Event{
			Action: "show_text",
		},
		Value: text,
	}
	return b.self
}

func (b *Base[T]) ShowItem(id string, count int32) T {
	b.HoverEvent = HoverEventShowItem{
		Event: Event{
			Action: "show_item",
		},
		Id:    id,
		Count: count,
	}
	return b.self
}

// SetInsertion insert a text in chat input when the component is shift-clicked.
func (b *Base[T]) SetInsertion(text string) T {
	b.Insertion = text
	return b.self
}
