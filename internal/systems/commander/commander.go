package commander

import (
	"context"
	"fmt"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
)

type Dispatcher struct {
	Root *Node
}

type ParsedCommand struct {
	Source  *CommandSource
	Reader  *CommandReader
	Nodes   []ParsedNode
	Args    ParsedArgs
	Command Command
	Forks   bool
	Errors  []*CommandParseError
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		Root: &Node{
			Kind:     RootNode,
			Children: make(map[string]*Node),
		},
	}
}

func (d *Dispatcher) Register(nodes ...*Node) {
	for _, n := range nodes {
		if n.Kind != LiteralNode {
			panic(fmt.Errorf("commander: root command '%s' must be a Literal, got %d", n.Name, n.Kind))
		}
		d.Root.Children[n.Name] = n
	}
}

func (d *Dispatcher) Resolve(name string) *Node {
	if child, ok := d.Root.Children[name]; ok {
		return child
	}
	return nil
}

func (d *Dispatcher) Parse(src *CommandSource, input string) *ParsedCommand {
	reader := NewCommandReader(input)
	result := &ParsedCommand{
		Source: src,
		Reader: reader,
		Args:   make(ParsedArgs),
	}

	d.parseNodes(d.Root, reader, result)
	return result
}

func (d *Dispatcher) parseNodes(node *Node, reader *CommandReader, result *ParsedCommand) {
	for reader.CanRead() && reader.Peek() == ' ' {
		reader.Skip()
	}

	if !reader.CanRead() {
		if node.Run != nil {
			result.Command = node.Run
		}
		return
	}

	start := reader.Cursor()
	token := reader.PeekWord()
	if child, ok := node.Children[token]; ok && child.Kind == LiteralNode {
		if child.PermissionLevel > 0 && !result.Source.HasPermission(child.PermissionLevel) {
			result.Errors = append(result.Errors, NewParseError(
				tc.Text("permission level too low"),
				reader.Input(), reader.Cursor(),
			))
			return
		}

		reader.ReadWord()
		end := reader.Cursor()
		result.Nodes = append(result.Nodes, ParsedNode{
			Node:  child,
			Range: StringRange{Start: start, End: end},
		})

		if child.Redirect != nil {
			if child.Fork {
				result.Forks = true
			}
			d.parseNodes(child.Redirect, reader, result)
			return
		}

		d.parseNodes(child, reader, result)
		return
	}

	for _, child := range node.Children {
		if child.Kind != ArgumentNode {
			continue
		}
		if child.PermissionLevel > 0 && !result.Source.HasPermission(child.PermissionLevel) {
			continue
		}

		reader.SetCursor(start)
		val, err := child.Parser.Parse(reader)
		if err != nil {
			var pe *CommandParseError
			if e, ok := err.(*CommandParseError); ok {
				pe = e
			} else {
				pe = NewParseError(tc.Text(err.Error()), reader.Input(), reader.Cursor())
			}
			result.Errors = append(result.Errors, pe)
			reader.SetCursor(start)
			continue
		}

		end := reader.Cursor()
		result.Args[child.Name] = val
		result.Nodes = append(result.Nodes, ParsedNode{
			Node:  child,
			Range: StringRange{Start: start, End: end},
		})

		if child.Redirect != nil {
			if child.Fork {
				result.Forks = true
			}
			d.parseNodes(child.Redirect, reader, result)
			return
		}

		d.parseNodes(child, reader, result)
		return
	}

	if len(result.Errors) == 0 {
		result.Errors = append(result.Errors, NewParseError(
			tc.Translatable(mcdata.CommandUnknownCommand),
			reader.Input(), reader.Cursor(),
		))
	}
}

func (d *Dispatcher) Execute(ctx context.Context, parsed *ParsedCommand) (*CommandResult, error) {
	if parsed.Command == nil {
		if len(parsed.Errors) > 0 {
			return nil, parsed.Errors[0]
		}
		return nil, NewParseError(
			tc.Translatable(mcdata.CommandUnknownCommand),
			parsed.Reader.Input(), parsed.Reader.Cursor(),
		)
	}

	sources := []*CommandSource{parsed.Source}
	if parsed.Forks {
		for _, pn := range parsed.Nodes {
			if pn.Node.Fork && pn.Node.RedirectModifier != nil {
				var nextSources []*CommandSource
				for _, src := range sources {
					derived, err := pn.Node.RedirectModifier(ctx, src)
					if err != nil {
						return nil, err
					}
					nextSources = append(nextSources, derived...)
				}
				sources = nextSources
				if len(sources) == 0 {
					// Early termination — branch count is zero.
					return &CommandResult{Success: 0, Result: 0}, nil
				}
			}
		}
	}

	aggregate := &CommandResult{}
	var lastErr error
	for _, src := range sources {
		if err := ctx.Err(); err != nil {
			return aggregate, err // context cancelled
		}

		res, err := parsed.Command(ctx, src, parsed.Args)
		if err != nil {
			lastErr = err
			continue
		}
		if res != nil {
			aggregate.Success += res.Success
			aggregate.Result = res.Result
		}
	}
	if aggregate.Success == 0 && lastErr != nil {
		return aggregate, lastErr
	}

	return aggregate, nil
}

func (d *Dispatcher) ExecuteInput(ctx context.Context, src *CommandSource, input string) (*CommandResult, error) {
	parsed := d.Parse(src, input)
	return d.Execute(ctx, parsed)
}

func (d *Dispatcher) FlattenGraph() ([]*Node, map[*Node]int) {
	var nodes []*Node
	indices := make(map[*Node]int)

	var walk func(n *Node)
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
