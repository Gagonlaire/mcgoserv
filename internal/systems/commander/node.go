package commander

type NodeType int
type SuggestionType string

const (
	RootNode NodeType = iota
	LiteralNode
	ArgumentNode
)

const (
	// implement suggest ask server
	SuggestNothing            SuggestionType = "" // default
	SuggestAskServer          SuggestionType = "ask_server"
	SuggestAllRecipes         SuggestionType = "all_recipes"
	SuggestAvailableSounds    SuggestionType = "available_sounds"
	SuggestSummonableEntities SuggestionType = "summonable_entities"
)

type Node struct {
	Kind       NodeType
	Name       string
	Children   map[string]*Node
	Run        Command
	Parser     ArgumentParser
	Suggestion SuggestionType
	Restricted bool
	Redirect   *Node
}

func Literal(name string) *Node {
	return &Node{
		Kind:     LiteralNode,
		Name:     name,
		Children: make(map[string]*Node),
	}
}

func Argument(name string, parser ArgumentParser) *Node {
	return &Node{
		Kind:     ArgumentNode,
		Name:     name,
		Parser:   parser,
		Children: make(map[string]*Node),
	}
}

func (n *Node) Connect(children ...*Node) *Node {
	if n.Children == nil {
		n.Children = make(map[string]*Node)
	}
	for _, child := range children {
		// todo: this should handle merge
		n.Children[child.Name] = child
	}
	return n
}

func (n *Node) RedirectTo(target *Node) *Node {
	if target.Kind != LiteralNode {
		panic("Redirect target must be a literal node")
	}

	n.Redirect = target
	return n
}

func (n *Node) Executes(cmd Command) *Node {
	n.Run = cmd
	return n
}

func (n *Node) SetSuggestion(suggestType SuggestionType) *Node {
	if n.Kind != ArgumentNode {
		panic("Only argument nodes can have suggestions")
	}
	n.Suggestion = suggestType
	return n
}

func (n *Node) SetRestricted() *Node {
	// todo: maybe use a op level instead and set the restricted if level is above 0?
	n.Restricted = true
	return n
}

func (n *Node) GetFlags() byte {
	flags := byte(0)

	flags |= byte(n.Kind) & 0x03
	if n.Run != nil {
		flags |= 0x04
	}
	if n.Redirect != nil {
		flags |= 0x08
	}
	if n.Suggestion != "" {
		flags |= 0x10
	}
	if n.Restricted {
		flags |= 0x20
	}

	return flags
}
