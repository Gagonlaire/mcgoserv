package commander

import (
	"fmt"
	"runtime"
	"strings"
)

// Build constructs a single linear command path from a syntax string and a list of parsers.
func Build(syntax string, parsers ...ArgumentParser) *Builder {
	tokens := tokenizeSyntax(syntax)
	wantParsers := 0
	for _, t := range tokens {
		if t.consumesParser() {
			wantParsers++
		}
	}
	if wantParsers != len(parsers) {
		panic(fmt.Errorf(
			"commander: Build(%q): expected %d parsers, got %d",
			syntax, wantParsers, len(parsers),
		))
	}
	_, file, line, _ := runtime.Caller(1)
	return &Builder{
		syntax:   syntax,
		tokens:   tokens,
		parsers:  parsers,
		callSite: fmt.Sprintf("%s:%d", file, line),
	}
}

type Builder struct {
	syntax   string
	tokens   []syntaxToken
	parsers  []ArgumentParser
	aliases  []string
	perm     int
	desc     string
	callSite string
}

func (b *Builder) Requires(level int) *Builder {
	b.perm = level
	return b
}

func (b *Builder) Description(desc string) *Builder {
	b.desc = desc
	return b
}

func (b *Builder) Aliases(names ...string) *Builder {
	b.aliases = append(b.aliases, names...)
	return b
}

func (b *Builder) Executes(cmd Command) *Node {
	for _, t := range b.tokens {
		if t.kind == tokChoice && t.name == "" {
			panic(fmt.Errorf(
				"commander: Build(%q): anonymous choice requires ExecutesEach; either name the choice (<name: a|b|c>) or pass one fn per branch",
				b.syntax,
			))
		}
	}
	return b.attach(cmd, nil)
}

func (b *Builder) ExecutesEach(fns ...Command) *Node {
	if len(fns) == 0 {
		panic(fmt.Errorf("commander: Build(%q): ExecutesEach requires at least one fn", b.syntax))
	}
	return b.attach(nil, fns)
}

func (b *Builder) attach(single Command, perShape []Command) *Node {
	if b.tokens[0].kind != tokLiteral {
		panic(fmt.Errorf("commander: Build(%q): first token must be a literal (root command)", b.syntax))
	}

	d := registrationDispatcher.get()
	if d == nil {
		panic("commander: Build() called outside a Dispatcher.RegisterBuilders frame")
	}

	root := b.ensureRootLiteral(d, b.tokens[0].name)
	if b.desc != "" {
		root.Description = b.desc
	}
	for _, alias := range b.aliases {
		b.ensureAlias(d, alias, root)
	}
	leafShapes := b.expandLeafShapes()

	var fns []Command
	if single != nil {
		fns = make([]Command, len(leafShapes))
		for i := range fns {
			fns[i] = single
		}
	} else {
		if len(perShape) != len(leafShapes) {
			panic(fmt.Errorf(
				"commander: Build(%q): ExecutesEach expected %d fn(s) (one per leaf shape), got %d",
				b.syntax, len(leafShapes), len(perShape),
			))
		}
		fns = perShape
	}

	var firstLeaf *Node
	for i, shape := range leafShapes {
		leaf := b.attachShape(root, shape, fns[i])
		if firstLeaf == nil {
			firstLeaf = leaf
		}
	}
	return firstLeaf
}

type leafShape struct {
	steps []shapeStep
}

type shapeStep struct {
	kind      syntaxTokenKind
	name      string
	parserIdx int
	bindKey   string
}

func (b *Builder) expandLeafShapes() []leafShape {
	parserIdx := 0
	var head []shapeStep
	var optionalSteps []shapeStep

	for i := 1; i < len(b.tokens); i++ {
		t := b.tokens[i]
		switch t.kind {
		case tokLiteral:
			head = append(head, shapeStep{kind: tokLiteral, name: t.name, parserIdx: -1})
		case tokRequired:
			head = append(head, shapeStep{kind: tokRequired, name: t.name, parserIdx: parserIdx})
			parserIdx++
		case tokVariadic:
			head = append(head, shapeStep{kind: tokVariadic, name: t.name, parserIdx: parserIdx})
			parserIdx++
		case tokOptional:
			optionalSteps = append(optionalSteps, shapeStep{kind: tokOptional, name: t.name, parserIdx: parserIdx})
			parserIdx++
		case tokChoice:
			head = append(head, shapeStep{kind: tokChoice, name: t.name, parserIdx: i})
		}
	}

	baseShapes := []leafShape{{steps: append([]shapeStep(nil), head...)}}
	for _, opt := range optionalSteps {
		prev := baseShapes[len(baseShapes)-1].steps
		extended := make([]shapeStep, len(prev)+1)
		copy(extended, prev)
		extended[len(extended)-1] = opt
		baseShapes = append(baseShapes, leafShape{steps: extended})
	}

	shapes := baseShapes
	for {
		var next []leafShape
		fanOut := false
		for _, sh := range shapes {
			idx := -1
			for i, st := range sh.steps {
				if st.kind == tokChoice {
					idx = i
					break
				}
			}
			if idx == -1 {
				next = append(next, sh)
				continue
			}
			fanOut = true
			choiceTok := b.tokens[sh.steps[idx].parserIdx]
			for _, branch := range choiceTok.choices {
				steps := make([]shapeStep, len(sh.steps))
				copy(steps, sh.steps)
				steps[idx] = shapeStep{
					kind:      tokLiteral,
					name:      branch,
					parserIdx: -1,
					bindKey:   choiceTok.name,
				}
				next = append(next, leafShape{steps: steps})
			}
		}
		shapes = next
		if !fanOut {
			break
		}
	}
	return shapes
}

func (b *Builder) ensureAlias(d *Dispatcher, alias string, target *Node) {
	if !isLiteralName(alias) {
		panic(fmt.Errorf("commander: Build(%q): alias %q is not a valid literal name", b.syntax, alias))
	}
	if alias == target.Name {
		panic(fmt.Errorf("commander: Build(%q): alias %q is identical to root command", b.syntax, alias))
	}
	for _, c := range d.Root.Children {
		if c.Kind != LiteralNode || c.Name != alias {
			continue
		}
		if c.Redirect == nil {
			panic(fmt.Errorf(
				"commander: Build(%q): alias %q collides with existing root command at %s",
				b.syntax, alias, callSiteOf(c),
			))
		}
		if c.Redirect != target {
			panic(fmt.Errorf(
				"commander: Build(%q): alias %q already redirects to %q at %s",
				b.syntax, alias, c.Redirect.Name, callSiteOf(c),
			))
		}
		return
	}
	n := Literal(alias)
	n.RegisteredAt = b.callSite
	n.Redirect = target
	d.Root.Children = append(d.Root.Children, n)
}

func (b *Builder) ensureRootLiteral(d *Dispatcher, name string) *Node {
	for _, c := range d.Root.Children {
		if c.Kind == LiteralNode && c.Name == name {
			return c
		}
	}
	root := Literal(name)
	root.RegisteredAt = b.callSite
	d.Root.Children = append(d.Root.Children, root)
	return root
}

func (b *Builder) attachShape(root *Node, shape leafShape, cmd Command) *Node {
	cur := root
	for i, step := range shape.steps {
		isLast := i == len(shape.steps)-1
		var next *Node
		switch step.kind {
		case tokLiteral:
			if step.bindKey != "" {
				next = b.matchOrAddChoiceLiteral(cur, step.name, step.bindKey)
			} else {
				next = b.matchOrAddLiteral(cur, step.name)
			}
		case tokRequired, tokVariadic, tokOptional:
			next = b.matchOrAddArgument(cur, step.name, b.parsers[step.parserIdx])
		}
		if isLast {
			if next.Run != nil {
				panic(fmt.Errorf(
					"commander: Build(%q): leaf already has an execute fn registered at %s",
					b.syntax, callSiteOf(next),
				))
			}
			next.Run = cmd
			next.PermissionLevel = b.perm
		}
		cur = next
	}
	return cur
}

func callSiteOf(n *Node) string {
	if n.RegisteredAt == "" {
		return "<unknown call site>"
	}
	return n.RegisteredAt
}

func (b *Builder) matchOrAddLiteral(parent *Node, name string) *Node {
	for _, c := range parent.Children {
		if c.Kind == LiteralNode && c.Name == name {
			return c
		}
	}
	n := Literal(name)
	n.RegisteredAt = b.callSite
	parent.Children = append(parent.Children, n)
	return n
}

func (b *Builder) matchOrAddChoiceLiteral(parent *Node, branch, bindKey string) *Node {
	for _, c := range parent.Children {
		if c.Kind == LiteralNode && c.Name == branch {
			if c.BindKey != "" && c.BindKey != bindKey {
				panic(fmt.Errorf(
					"commander: Build(%q): choice branch %q already binds to %q, cannot rebind to %q (added at %s)",
					b.syntax, branch, c.BindKey, bindKey, callSiteOf(c),
				))
			}
			c.BindKey = bindKey
			return c
		}
	}
	n := Literal(branch)
	n.RegisteredAt = b.callSite
	n.BindKey = bindKey
	parent.Children = append(parent.Children, n)
	return n
}

func (b *Builder) matchOrAddArgument(parent *Node, name string, parser ArgumentParser) *Node {
	for _, c := range parent.Children {
		if c.Kind == ArgumentNode && c.Name == name {
			if c.Parser != parser {
				panic(fmt.Errorf(
					"commander: Build(%q): argument <%s> at parent %q already exists with a different parser (added at %s)",
					b.syntax, name, parent.Name, callSiteOf(c),
				))
			}
			return c
		}
	}
	n := Argument(name, parser)
	n.RegisteredAt = b.callSite
	parent.Children = append(parent.Children, n)
	return n
}

type dispatcherRegistration struct {
	stack []*Dispatcher
}

func (r *dispatcherRegistration) push(d *Dispatcher) { r.stack = append(r.stack, d) }

func (r *dispatcherRegistration) pop() { r.stack = r.stack[:len(r.stack)-1] }

func (r *dispatcherRegistration) get() *Dispatcher {
	if len(r.stack) == 0 {
		return nil
	}
	return r.stack[len(r.stack)-1]
}

var registrationDispatcher = &dispatcherRegistration{}

func (d *Dispatcher) RegisterBuilders(fn func()) {
	registrationDispatcher.push(d)
	defer registrationDispatcher.pop()
	fn()
}

type syntaxTokenKind int

const (
	tokLiteral syntaxTokenKind = iota
	tokRequired
	tokOptional
	tokChoice
	tokVariadic
)

type syntaxToken struct {
	kind    syntaxTokenKind
	name    string
	choices []string
}

func (t syntaxToken) consumesParser() bool {
	switch t.kind {
	case tokRequired, tokOptional, tokVariadic:
		return true
	default:
		return false
	}
}

func tokenizeSyntax(syntax string) []syntaxToken {
	s := strings.TrimSpace(syntax)
	s = strings.TrimPrefix(s, "/")
	if s == "" {
		panic(fmt.Errorf("commander: Build(%q): empty syntax", syntax))
	}
	parts := splitTopLevel(s)

	var out []syntaxToken
	sawOptional := false
	for i, part := range parts {
		tok := parseToken(syntax, part)
		isLast := i == len(parts)-1
		switch tok.kind {
		case tokVariadic:
			if !isLast {
				panic(fmt.Errorf(
					"commander: Build(%q): variadic <%s ...> must be the last token",
					syntax, tok.name,
				))
			}
		case tokRequired, tokLiteral, tokChoice:
			if sawOptional && tok.kind != tokChoice {
				panic(fmt.Errorf(
					"commander: Build(%q): %q appears after an optional; only optionals may follow optionals",
					syntax, part,
				))
			}
			if tok.kind == tokChoice && !isLast {
				if tok.name == "" {
					panic(fmt.Errorf(
						"commander: Build(%q): anonymous choice %q must be the last token (otherwise name it: <name: %s>)",
						syntax, part, strings.Join(tok.choices, "|"),
					))
				}
			}
		case tokOptional:
			sawOptional = true
		}
		out = append(out, tok)
	}
	if out[0].kind != tokLiteral {
		panic(fmt.Errorf("commander: Build(%q): first token must be a bare literal", syntax))
	}
	return out
}

func splitTopLevel(s string) []string {
	var parts []string
	depth := 0
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '<', '[':
			depth++
			cur.WriteByte(c)
		case '>', ']':
			depth--
			cur.WriteByte(c)
		case ' ':
			if depth == 0 {
				if cur.Len() > 0 {
					parts = append(parts, cur.String())
					cur.Reset()
				}
				continue
			}
			cur.WriteByte(c)
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

func parseToken(syntax, part string) syntaxToken {
	if part == "" {
		panic(fmt.Errorf("commander: Build(%q): empty token", syntax))
	}
	switch part[0] {
	case '<':
		if part[len(part)-1] != '>' {
			panic(fmt.Errorf("commander: Build(%q): unterminated <...> in %q", syntax, part))
		}
		inner := part[1 : len(part)-1]
		return parseAngleToken(syntax, inner, false)
	case '[':
		if part[len(part)-1] != ']' {
			panic(fmt.Errorf("commander: Build(%q): unterminated [...] in %q", syntax, part))
		}
		inner := strings.TrimSpace(part[1 : len(part)-1])
		if len(inner) < 2 || inner[0] != '<' || inner[len(inner)-1] != '>' {
			panic(fmt.Errorf("commander: Build(%q): optional %q must wrap an angle placeholder, e.g. [<name>]", syntax, part))
		}
		t := parseAngleToken(syntax, inner[1:len(inner)-1], true)
		if t.kind == tokVariadic {
			panic(fmt.Errorf("commander: Build(%q): optional variadic [<%s ...>] is not supported", syntax, t.name))
		}
		if t.kind == tokChoice {
			panic(fmt.Errorf("commander: Build(%q): optional choice %q is not supported; use ExecutesEach or a named choice", syntax, part))
		}
		return t
	default:
		if !isLiteralName(part) {
			panic(fmt.Errorf("commander: Build(%q): %q is not a valid literal name", syntax, part))
		}
		return syntaxToken{kind: tokLiteral, name: part}
	}
}

func parseAngleToken(syntax, inner string, fromOptional bool) syntaxToken {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		panic(fmt.Errorf("commander: Build(%q): empty placeholder", syntax))
	}

	if strings.HasSuffix(inner, "...") {
		name := strings.TrimSpace(strings.TrimSuffix(inner, "..."))
		if name == "" {
			panic(fmt.Errorf("commander: Build(%q): variadic must have a name", syntax))
		}
		if !isPlaceholderName(name) {
			panic(fmt.Errorf("commander: Build(%q): %q is not a valid variadic name", syntax, name))
		}
		return syntaxToken{kind: tokVariadic, name: name}
	}

	colonIdx := strings.Index(inner, ":")
	pipeIdx := strings.Index(inner, "|")
	if colonIdx >= 0 && (pipeIdx == -1 || colonIdx < pipeIdx) {
		name := strings.TrimSpace(inner[:colonIdx])
		body := strings.TrimSpace(inner[colonIdx+1:])
		if !isPlaceholderName(name) {
			panic(fmt.Errorf("commander: Build(%q): %q is not a valid choice name", syntax, name))
		}
		choices := splitChoices(syntax, body)
		t := syntaxToken{kind: tokChoice, name: name, choices: choices}
		_ = fromOptional
		return t
	}
	if pipeIdx >= 0 {
		choices := splitChoices(syntax, inner)
		return syntaxToken{kind: tokChoice, name: "", choices: choices}
	}

	if !isPlaceholderName(inner) {
		panic(fmt.Errorf("commander: Build(%q): %q is not a valid placeholder name", syntax, inner))
	}
	if fromOptional {
		return syntaxToken{kind: tokOptional, name: inner}
	}
	return syntaxToken{kind: tokRequired, name: inner}
}

func splitChoices(syntax, body string) []string {
	raw := strings.Split(body, "|")
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, r := range raw {
		c := strings.TrimSpace(r)
		if c == "" {
			panic(fmt.Errorf("commander: Build(%q): empty choice branch", syntax))
		}
		if !isLiteralName(c) {
			panic(fmt.Errorf("commander: Build(%q): %q is not a valid choice branch", syntax, c))
		}
		if _, dup := seen[c]; dup {
			panic(fmt.Errorf("commander: Build(%q): duplicate choice branch %q", syntax, c))
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	if len(out) < 2 {
		panic(fmt.Errorf("commander: Build(%q): choice must have at least two branches", syntax))
	}
	return out
}

func isLiteralName(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-'
		if !ok {
			return false
		}
	}
	return true
}

func isPlaceholderName(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
		if !ok {
			return false
		}
	}
	return true
}
