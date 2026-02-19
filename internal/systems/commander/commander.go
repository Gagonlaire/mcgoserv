package commander

import (
	"context"
	"fmt"
	"strings"
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
	Parse(reader *strings.Reader) (interface{}, error)
	ID() string
	// Serialize(buf *bytes.Buffer) todo: implement later
}

type Command func(ctx *CommandContext) (string, error)

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

func (d *Commander) Execute(ctx context.Context, input string) (string, error) {
	reader := strings.NewReader(input)
	cmdCtx := &CommandContext{
		Context: ctx,
		Args:    make(map[string]interface{}),
		Input:   input,
	}
	current := d.Root

	for reader.Len() > 0 {
		if err := cmdCtx.Err(); err != nil {
			return "", err
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
					reader.Seek(int64(len(input)-startLen), 0)

					val, err := child.Parser.Parse(reader)
					if err == nil {
						cmdCtx.Args[child.Name] = val
						found = child
						break
					} else {
						return "", &SyntaxError{
							Code:     InvalidArgument,
							NodeName: child.Name,
							Token:    token,
							Input:    input,
							Cursor:   len(input) - reader.Len(),
						}
					}
				}
			}
		}

		if found == nil {
			reader.Seek(int64(len(input)-startLen), 0)
			badToken := readWord(reader)
			cursor := len(input) - reader.Len()

			return "", &SyntaxError{
				Code:     UnknownCommand,
				NodeName: current.Name,
				Token:    badToken,
				Input:    input,
				Cursor:   cursor,
			}
		}

		current = found
		if current.Redirect != nil {
			current = current.Redirect
		}

		skipWhitespace(reader)
	}

	if current.Run == nil {
		return "", &SyntaxError{
			Code:     IncompleteCommand,
			NodeName: current.Name,
			Input:    input,
			Cursor:   len(input),
		}
	}

	return current.Run(cmdCtx)
}

func peekWord(r *strings.Reader) string {
	start, _ := r.Seek(0, 1)
	word := readWord(r)
	r.Seek(start, 0)
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
