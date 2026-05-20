package commander

import (
	"io"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
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
	Children         []*Node
	Run              Command
	SuggestFn        SuggestFunc
	Redirect         *Node
	RedirectModifier RedirectModifier
	Name             string
	Suggestion       SuggestionType
	Kind             NodeType
	PermissionLevel  int
	IsFork           bool
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
		Kind: LiteralNode,
		Name: name,
	}
}

func Argument(name string, parser ArgumentParser) *Node {
	return &Node{
		Kind:   ArgumentNode,
		Name:   name,
		Parser: parser,
	}
}

func (n *Node) Connect(children ...*Node) *Node {
	n.Children = append(n.Children, children...)
	return n
}

func (n *Node) findLiteralChild(name string) *Node {
	for _, c := range n.Children {
		if c.Kind == LiteralNode && c.Name == name {
			return c
		}
	}
	return nil
}

// Redirects points this node at target. With no Modify() it is a pure alias
func (n *Node) Redirects(target *Node) *Node {
	if target.Kind != LiteralNode {
		panic("commander: redirect target must be a literal node")
	}
	n.Redirect = target
	return n
}

// Modify attaches a RedirectModifier to this node. The modifier runs at
// execution time, transforming the current source set into the next set
func (n *Node) Modify(fn RedirectModifier) *Node {
	n.RedirectModifier = fn
	return n
}

// Fork marks this node as a fan-out point. Downstream execution runs once
// per derived source and success counts sum across branches.
func (n *Node) Fork() *Node {
	n.IsFork = true
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
