package commander

import (
	"context"
	"fmt"
	"maps"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
)

type Dispatcher struct {
	Root *Node
}

type ParsedCommand struct {
	Source  *CommandSource
	Reader  *CommandReader
	Args    ParsedArgs
	Signed  *SignedData
	Command Command
	Nodes   []ParsedNode
	Errors  []*CommandParsingError
	Forks   bool
}

type SuggestionContext struct {
	Node   *Node
	Start  int
	Length int
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		Root: &Node{Kind: RootNode},
	}
}

func (d *Dispatcher) Register(nodes ...*Node) {
	for _, n := range nodes {
		if n.Kind != LiteralNode {
			panic(fmt.Errorf("commander: root command '%s' must be a Literal, got %d", n.Name, n.Kind))
		}
		d.Root.Children = append(d.Root.Children, n)
	}
}

func (d *Dispatcher) Resolve(name string) *Node {
	return d.Root.findLiteralChild(name)
}

// deepestError picks the parsing error whose cursor reached furthest into the input.
func deepestError(errs []*CommandParsingError) *CommandParsingError {
	best := errs[0]
	for _, e := range errs[1:] {
		if e.cursor > best.cursor {
			best = e
		}
	}
	return best
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

// parseSnapshot captures everything parseNodes mutates, so we can rollback
type parseSnapshot struct {
	args    ParsedArgs
	nodes   []ParsedNode
	errors  []*CommandParsingError
	command Command
	forks   bool
	cursor  int
}

func snapshotResult(result *ParsedCommand, reader *CommandReader) parseSnapshot {
	return parseSnapshot{
		args:    maps.Clone(result.Args),
		nodes:   append([]ParsedNode(nil), result.Nodes...),
		errors:  append([]*CommandParsingError(nil), result.Errors...),
		command: result.Command,
		forks:   result.Forks,
		cursor:  reader.Cursor(),
	}
}

func restoreSnapshot(result *ParsedCommand, reader *CommandReader, s parseSnapshot) {
	result.Args = s.args
	result.Nodes = s.nodes
	result.Errors = s.errors
	result.Command = s.command
	result.Forks = s.forks
	reader.SetCursor(s.cursor)
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
	if literal := node.findLiteralChild(token); literal != nil && result.Source.HasPermission(literal.PermissionLevel) {
		reader.ReadWord()
		end := reader.Cursor()
		result.Nodes = append(result.Nodes, ParsedNode{
			Node:  literal,
			Range: StringRange{Start: start, End: end},
		})

		if literal.Redirect != nil {
			if literal.Fork {
				result.Forks = true
			}
			d.parseNodes(literal.Redirect, reader, result)
			return
		}

		d.parseNodes(literal, reader, result)
		return
	}

	baseline := snapshotResult(result, reader)
	bestSnapshot := baseline
	bestSnapshot.cursor = -1
	var attemptErrors []*CommandParsingError

	for _, child := range node.Children {
		if child.Kind != ArgumentNode {
			continue
		}
		if !result.Source.HasPermission(child.PermissionLevel) {
			continue
		}

		restoreSnapshot(result, reader, baseline)
		reader.SetCursor(start)

		val, err := child.Parser.Parse(reader)
		if err != nil {
			attemptErrors = append(attemptErrors, err.(*CommandParsingError))
			continue
		}

		if err := reader.ExpectSeparator(); err != nil {
			attemptErrors = append(attemptErrors, err.(*CommandParsingError))
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
		} else {
			d.parseNodes(child, reader, result)
		}

		candidate := snapshotResult(result, reader)
		attemptErrors = append(attemptErrors, candidate.errors[len(baseline.errors):]...)

		if bestSnapshot.cursor < 0 || candidate.cursor > bestSnapshot.cursor {
			bestSnapshot = candidate
		}
	}

	if bestSnapshot.cursor >= 0 {
		restoreSnapshot(result, reader, bestSnapshot)
		return
	}

	// All argument children failed, surface their errors so deepestError can pick.
	restoreSnapshot(result, reader, baseline)
	if len(attemptErrors) > 0 {
		result.Errors = append(result.Errors, attemptErrors...)
		return
	}
	result.Errors = append(result.Errors, NewParsingError(
		reader,
		tc.Translatable(mcdata.CommandUnknownCommand),
	))
}

func (d *Dispatcher) Execute(ctx context.Context, parsed *ParsedCommand) (*CommandResult, error) {
	if parsed.Command == nil {
		if len(parsed.Errors) > 0 {
			best := deepestError(parsed.Errors)
			// If the parsing never got past the root node, we consider it
			// an execution error instead of a parsing error, to match vanilla's behavior
			if len(parsed.Nodes) == 0 {
				return nil, NewExecutionErrorFromParsing(best)
			}
			return nil, best
		}
		return nil, NewExecutionError(
			parsed.Reader,
			tc.Translatable(mcdata.CommandUnknownCommand),
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
			Signed: parsed.Signed,
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

func (d *Dispatcher) ParseSigned(src *CommandSource, input string, signed *SignedData) *ParsedCommand {
	parsed := d.Parse(src, input)
	parsed.Signed = signed
	return parsed
}

func (d *Dispatcher) ExecuteSignedInput(ctx context.Context, src *CommandSource, input string, signed *SignedData) (*CommandResult, error) {
	parsed := d.ParseSigned(src, input, signed)
	return d.Execute(ctx, parsed)
}

func (d *Dispatcher) ParseForSuggestion(src *CommandSource, input string) *SuggestionContext {
	reader := NewCommandReader(input)
	return d.parseNodeForSuggestion(d.Root, reader, src)
}

func (d *Dispatcher) parseNodeForSuggestion(node *Node, reader *CommandReader, src *CommandSource) *SuggestionContext {
	for reader.CanRead() && reader.Peek() == ' ' {
		reader.Skip()
	}

	argChildren := func() []*Node {
		var args []*Node
		for _, child := range node.Children {
			if child.Kind == ArgumentNode && src.HasPermission(child.PermissionLevel) {
				args = append(args, child)
			}
		}
		return args
	}

	if !reader.CanRead() {
		for _, child := range argChildren() {
			if child.Suggestion == SuggestAskServer {
				return &SuggestionContext{
					Node:   child,
					Start:  reader.Cursor(),
					Length: 0,
				}
			}
		}
		return nil
	}

	start := reader.Cursor()
	token := reader.PeekWord()

	if literal := node.findLiteralChild(token); literal != nil && src.HasPermission(literal.PermissionLevel) {
		reader.ReadWord()

		if !reader.CanRead() {
			return nil
		}

		if literal.Redirect != nil {
			return d.parseNodeForSuggestion(literal.Redirect, reader, src)
		}
		return d.parseNodeForSuggestion(literal, reader, src)
	}

	for _, child := range argChildren() {
		reader.SetCursor(start)
		_, err := child.Parser.Parse(reader)
		sepErr := reader.ExpectSeparator()
		canRead := reader.CanRead()

		failed := err != nil || sepErr != nil || !canRead

		if failed {
			if child.Suggestion == SuggestAskServer {
				// suggestion covers from start of this argument to end of input
				return &SuggestionContext{
					Node:   child,
					Start:  start,
					Length: reader.TotalLength() - start,
				}
			}
			if err != nil || sepErr != nil {
				continue
			}
			return nil
		}

		if child.Redirect != nil {
			return d.parseNodeForSuggestion(child.Redirect, reader, src)
		}
		return d.parseNodeForSuggestion(child, reader, src)
	}

	return nil
}

func (d *Dispatcher) FlattenGraph(permissionLevel int) ([]*Node, map[*Node]int, map[*Node][]*Node) {
	var nodes []*Node
	indices := make(map[*Node]int)
	filteredChildren := make(map[*Node][]*Node)

	var walk func(n *Node)
	walk = func(n *Node) {
		if _, visited := indices[n]; visited {
			return
		}
		indices[n] = len(nodes)
		nodes = append(nodes, n)

		var children []*Node
		for _, child := range n.Children {
			if child.PermissionLevel > 0 && child.PermissionLevel > permissionLevel {
				continue
			}
			children = append(children, child)
		}

		filteredChildren[n] = children

		for _, child := range children {
			walk(child)
		}

		if n.Redirect != nil {
			walk(n.Redirect)
		}
	}
	walk(d.Root)

	return nodes, indices, filteredChildren
}
