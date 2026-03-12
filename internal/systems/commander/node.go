package commander

import (
	"fmt"
	"io"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
)

type NodeType int
type SuggestionType string

type ArgumentParser interface {
	Parse(reader *CommandReader) (any, error)
	ID() int
	WriteTo(w io.Writer) (n int64, err error)
}

type Command func(cc *CommandContext) (*CommandResult, error)

type SuggestFunc func(src *CommandSource, input string) []SuggestionEntry

type SuggestionEntry struct {
	Tooltip tc.Component
	Text    string
}

type ParsedArgs map[string]any

type Node struct {
	Parser           ArgumentParser
	Children         map[string]*Node
	Run              Command
	SuggestFn        SuggestFunc
	Redirect         *Node
	RedirectModifier RedirectModifier
	Name             string
	Suggestion       SuggestionType
	Kind             NodeType
	PermissionLevel  int
	Fork             bool
}

type ParsedNode struct {
	Node  *Node
	Range StringRange
}

type StringRange struct {
	Start int
	End   int
}

const (
	RootNode NodeType = iota
	LiteralNode
	ArgumentNode
)

const (
	NodeTypeMask           = 0x03
	IsExecutableMask       = 0x04
	HasRedirectMask        = 0x08
	HasSuggestionsTypeMask = 0x10
	IsRestrictedMask       = 0x20
)

const (
	SuggestNothing            SuggestionType = "" // default
	SuggestAskServer          SuggestionType = "ask_server"
	SuggestAllRecipes         SuggestionType = "all_recipes"
	SuggestAvailableSounds    SuggestionType = "available_sounds"
	SuggestSummonableEntities SuggestionType = "summonable_entities"
)

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
		if child.Kind == ArgumentNode {
			for _, existing := range n.Children {
				if existing.Kind == ArgumentNode {
					panic(fmt.Sprintf(
						"commander: node '%s' already has argument child '%s', cannot add '%s'",
						n.Name, existing.Name, child.Name,
					))
				}
			}
		}
		n.Children[child.Name] = child
	}
	return n
}

func (n *Node) RedirectTo(target *Node) *Node {
	if target.Kind != LiteralNode {
		panic("commander: redirect target must be a literal node")
	}
	n.Redirect = target
	return n
}

func (n *Node) ForkTo(target *Node, modifier RedirectModifier) *Node {
	n.Redirect = target
	n.RedirectModifier = modifier
	n.Fork = true
	return n
}

func (n *Node) Executes(cmd Command) *Node {
	n.Run = cmd
	return n
}

func (n *Node) SetSuggestion(suggestType SuggestionType) *Node {
	if n.Kind != ArgumentNode {
		panic("commander: only argument nodes can have suggestions")
	}
	n.Suggestion = suggestType
	return n
}

func (n *Node) ServerSuggestion(fn SuggestFunc) *Node {
	n.SetSuggestion(SuggestAskServer)
	n.SuggestFn = fn
	return n
}

func (n *Node) Requires(level int) *Node {
	n.PermissionLevel = level
	return n
}

func (n *Node) GetFlags() byte {
	flags := byte(n.Kind) & NodeTypeMask

	if n.Run != nil {
		flags |= IsExecutableMask
	}
	if n.Redirect != nil {
		flags |= HasRedirectMask
	}
	if n.Suggestion != "" {
		flags |= HasSuggestionsTypeMask
	}
	if n.PermissionLevel != 0 {
		flags |= IsRestrictedMask
	}

	return flags
}
