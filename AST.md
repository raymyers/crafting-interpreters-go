
This is an example of how to define a type-safe AST in Go using an ADT style.

```go
// Node is the base interface for all AST nodes
type Node interface {
	implNode()
}

// Expression variants
type Number struct {
	Value int
}
type Add struct {
	Left, Right Node
}
type Multiply struct {
	Left, Right Node
}

// Marker methods for interface implementation
func (Number) implNode()   {}
func (Add) implNode()      {}
func (Multiply) implNode() {}

// Evaluate recursively
func Eval(n Node) int {
	switch t := n.(type) {
	case Number:
		return t.Value
	case Add:
		return Eval(t.Left) + Eval(t.Right)
	case Multiply:
		return Eval(t.Left) * Eval(t.Right)
	}
	panic("unhandled node type")
}

```