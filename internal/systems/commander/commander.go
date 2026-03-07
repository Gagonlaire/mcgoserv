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
	Errors  []*CommandParsingError
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

		reader.SetCursor(start)
		val, err := child.Parser.Parse(reader)
		if err != nil {
			result.Errors = append(result.Errors, err.(*CommandParsingError))
			reader.SetCursor(start)
			continue
		}

		if err := reader.ExpectSeparator(); err != nil {
			result.Errors = append(result.Errors, err.(*CommandParsingError))
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
		result.Errors = append(result.Errors, NewParsingError(
			tc.Translatable(mcdata.CommandUnknownCommand),
			reader.Input(),
		))
	}
}

func (d *Dispatcher) Execute(ctx context.Context, parsed *ParsedCommand) (*CommandResult, error) {
	if parsed.Command == nil {
		if len(parsed.Errors) > 0 {
			// If the parsing never got past the root node, we consider it
			// an execution error instead of a parsing error, to match vanilla's behavior
			if len(parsed.Nodes) == 0 {
				return nil, NewExecutionError(parsed.Errors[0].component, parsed.Errors[0].input)
			}
			return nil, parsed.Errors[0]
		}
		return nil, NewExecutionError(
			tc.Translatable(mcdata.CommandUnknownCommand),
			parsed.Reader.Input(),
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
						return nil, AsCommandError(err)
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
			return aggregate, AsCommandError(err)
		}

		cc := &CommandContext{
			Ctx:    ctx,
			Source: src,
			Args:   parsed.Args,
		}
		res, err := parsed.Command(cc)
		if err != nil {
			lastErr = AsCommandError(err)
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
