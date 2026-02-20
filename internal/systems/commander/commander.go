package commander

import (
	"context"
	"fmt"
	"io"
	"strings"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
)

type Commander struct {
	Root *Node
}

type CommandContext struct {
	context.Context
	Args  map[string]interface{}
	Input string
}

type ArgumentParser interface {
	Parse(reader *strings.Reader) (interface{}, tc.Component)
	ID() int
	WriteTo(w io.Writer) (n int64, err error)
}

type Command func(ctx *CommandContext) tc.Component

func NewCommander() *Commander {
	return &Commander{
		Root: &Node{
			Kind:     RootNode,
			Children: make(map[string]*Node),
		},
	}
}

func (d *Commander) Register(nodes ...*Node) {
	for _, n := range nodes {
		if n.Kind != LiteralNode {
			panic(fmt.Errorf("root command '%s' must be a Literal, got %d", n.Name, n.Kind))
		}
		d.Root.Children[n.Name] = n
	}
}

func (d *Commander) Resolve(cmdName string) *Node {
	if child, ok := d.Root.Children[cmdName]; ok {
		return child
	}
	return nil
}

func (d *Commander) Execute(ctx context.Context, input string) tc.Component {
	reader := strings.NewReader(input)
	cmdCtx := &CommandContext{
		Context: ctx,
		Args:    make(map[string]interface{}),
		Input:   input,
	}
	current := d.Root

	for reader.Len() > 0 {
		if err := cmdCtx.Err(); err != nil {
			return nil
		}

		startLen := reader.Len()
		token := peekWord(reader)
		var found *Node

		if child, ok := current.Children[token]; ok && child.Kind == LiteralNode {
			readWord(reader)
			found = child
		} else {
			for _, child := range current.Children {
				if child.Kind == ArgumentNode {
					_, _ = reader.Seek(int64(len(input)-startLen), 0)

					val, err := child.Parser.Parse(reader)
					if err == nil {
						cmdCtx.Args[child.Name] = val
						found = child
						break
					} else {
						return err
					}
				}
			}
		}

		if found == nil {
			_, _ = reader.Seek(int64(len(input)-startLen), 0)
			badToken := readWord(reader)

			return tc.Translatable(mcdata.CommandUnknownCommand).AddExtra(
				tc.Container(
					tc.Text("\n"+badToken).SetUnderlined(true),
					tc.Translatable(mcdata.CommandContextHere),
				).SuggestCommand(badToken),
			).SetColor(tc.ColorRed)
		}

		current = found
		if current.Redirect != nil {
			current = current.Redirect
		}

		skipWhitespace(reader)
	}

	if current.Run == nil {
		start := len(input) - 10
		prefix := "..."

		if start <= 0 {
			start = 0
			prefix = ""
		}

		return tc.Translatable(mcdata.CommandUnknownCommand).AddExtra(
			tc.Container(
				tc.Text("\n"+prefix+input[start:]).SetColor(tc.ColorGray),
				tc.Translatable(mcdata.CommandContextHere),
			).SuggestCommand("/" + input),
		).SetColor(tc.ColorRed)
	}

	return current.Run(cmdCtx)
}

func (d *Commander) FlattenGraph() ([]*Node, map[*Node]int) {
	var nodes []*Node
	var walk func(n *Node)
	indices := make(map[*Node]int)

	walk = func(n *Node) {
		if _, visited := indices[n]; visited {
			return
		}

		indices[n] = len(nodes)
		nodes = append(nodes, n)
		for _, child := range n.Children {
			walk(child)
		}
		if n.Redirect != nil {
			walk(n.Redirect)
		}
	}
	walk(d.Root)

	return nodes, indices
}

func peekWord(r *strings.Reader) string {
	start, _ := r.Seek(0, 1)
	word := readWord(r)
	_, _ = r.Seek(start, 0)
	return word
}

func readWord(r *strings.Reader) string {
	var sb strings.Builder
	for r.Len() > 0 {
		ch, _, _ := r.ReadRune()
		if ch == ' ' {
			break
		}
		sb.WriteRune(ch)
	}
	return sb.String()
}

func skipWhitespace(r *strings.Reader) {
	for r.Len() > 0 {
		ch, _, _ := r.ReadRune()
		if ch != ' ' {
			_ = r.UnreadRune()
			break
		}
	}
}
