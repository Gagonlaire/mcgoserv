package commander

type NodeType int

const (
	RootNode NodeType = iota
	LiteralNode
	ArgumentNode
)

type Node struct {
	Kind     NodeType
	Name     string
	Children map[string]*Node
	Run      Command
	Parser   ArgumentParser
	Redirect *Node
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
