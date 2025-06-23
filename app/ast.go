package main

// Value represents a runtime value in the Lox language
type Value interface {
	implValue()
}

// StringValue represents a string literal
type StringValue struct {
	Val string
}

func (StringValue) implValue() {}

// NumberValue represents a numeric literal (float64)
type NumberValue struct {
	Val float64
}

func (NumberValue) implValue() {}

// BoolValue represents a boolean literal
type BoolValue struct {
	Val bool
}

func (BoolValue) implValue() {}

// NilValue represents the nil value
type NilValue struct{}

func (NilValue) implValue() {}

type FunValue struct {
	Val Fun
}

func (FunValue) implValue() {}

type ErrorValue struct {
	Message string
	Line    uint
}

func (ErrorValue) implValue() {}

// Expr represents an expression in the AST
type Expr interface {
	Accept(visitor ExprVisitor) Value
}

// ExprVisitor defines the visitor pattern for expressions
type ExprVisitor interface {
	VisitBinaryExpr(expr *Binary) Value
	VisitGroupingExpr(expr *Grouping) Value
	VisitLiteralExpr(expr *Literal) Value
	VisitUnaryExpr(expr *Unary) Value
	VisitVariableExpr(expr *Variable) Value
	VisitPrintStatement(expr *PrintStatement) Value
	VisitStatements(expr *Statements) Value
	VisitVarStatement(expr *VarStatement) Value
	VisitBlock(expr *Block) Value
	VisitIfStatement(expr *IfStatement) Value
	VisitWhileStatement(expr *WhileStatement) Value
	VisitForStatement(expr *ForStatement) Value
	VisitCallExpr(expr *Call) Value
	VisitFun(expr *Fun) Value
	VisitRecord(expr *Record) Value
	VisitEmptyRecord(expr *EmptyRecord) Value
	VisitList(expr *List) Value
	VisitAccess(expr *Access) Value
	VisitBuiltin(expr *Builtin) Value
	VisitUnion(expr *Union) Value
	VisitLambda(expr *Lambda) Value
	VisitMatch(expr *Match) Value
	VisitPerform(expr *Perform) Value
	VisitHandle(expr *Handle) Value
	VisitNamedRef(expr *NamedRef) Value
	VisitThunk(expr *Thunk) Value
	VisitSpread(expr *Spread) Value
	VisitDestructure(expr *Destructure) Value
	VisitSeq(expr *Seq) Value
}

// Binary represents a binary expression (e.g., 1 + 2)
type Binary struct {
	Left     Expr
	Operator Token
	Right    Expr
	Line     uint
}

func (b *Binary) Accept(visitor ExprVisitor) Value {
	return visitor.VisitBinaryExpr(b)
}

// Grouping represents a grouped expression (e.g., (1 + 2))
type Grouping struct {
	Expression Expr
	Line       uint
}

func (g *Grouping) Accept(visitor ExprVisitor) Value {
	return visitor.VisitGroupingExpr(g)
}

// Literal represents a literal value (e.g., 42, "hello", true)
type Literal struct {
	Value Value
	Line  uint
}

func (l *Literal) Accept(visitor ExprVisitor) Value {
	return visitor.VisitLiteralExpr(l)
}

// Unary represents a unary expression (e.g., -1, !true)
type Unary struct {
	Operator Token
	Right    Expr
	Line     uint
}

func (u *Unary) Accept(visitor ExprVisitor) Value {
	return visitor.VisitUnaryExpr(u)
}

// Variable represents a variable reference (e.g., x)
type Variable struct {
	Name Token
	Line uint
}

func (v *Variable) Accept(visitor ExprVisitor) Value {
	return visitor.VisitVariableExpr(v)
}

// PrintStatement (e.g., (1 + 2))
type PrintStatement struct {
	Expression Expr
	Line       uint
}

func (g *PrintStatement) Accept(visitor ExprVisitor) Value {
	return visitor.VisitPrintStatement(g)
}

// VarStatement (e.g., var a = 1)
type VarStatement struct {
	name       string
	Expression Expr
	Line       uint
}

func (g *VarStatement) Accept(visitor ExprVisitor) Value {
	return visitor.VisitVarStatement(g)
}

type Statements struct {
	Exprs []Expr
	Line  uint
}

func (g *Statements) Accept(visitor ExprVisitor) Value {
	return visitor.VisitStatements(g)
}

// Block represents a block statement (e.g., { statements })
type Block struct {
	Statements []Expr
	Line       uint
}

func (b *Block) Accept(visitor ExprVisitor) Value {
	return visitor.VisitBlock(b)
}

// IfStatement represents an if statement (e.g., if (condition) { then })
type IfStatement struct {
	Condition  Expr
	ThenBranch Expr
	ElseBranch Expr
	Line       uint
}

func (i *IfStatement) Accept(visitor ExprVisitor) Value {
	return visitor.VisitIfStatement(i)
}

// WhileStatement represents a while loop (e.g., while (condition) { body })
type WhileStatement struct {
	Condition Expr
	Body      Expr
	Line      uint
}

func (w *WhileStatement) Accept(visitor ExprVisitor) Value {
	return visitor.VisitWhileStatement(w)
}

type ForStatement struct {
	Initializer Expr
	Condition   Expr
	Increment   Expr
	Body        Expr
	Line        uint
}

func (w *ForStatement) Accept(visitor ExprVisitor) Value {
	return visitor.VisitForStatement(w)
}

// Call represents a function call expression (e.g., foo(1, 2, 3))
type Call struct {
	Callee    Expr
	Arguments []Expr
	Line      uint
}

func (c *Call) Accept(visitor ExprVisitor) Value {
	return visitor.VisitCallExpr(c)
}

type Fun struct {
	Name       string
	Parameters []string
	Block      Block
	Line       uint
}

func (c *Fun) Accept(visitor ExprVisitor) Value {
	return visitor.VisitFun(c)
}

// Record represents a record with fields (e.g., {name: "Alice", age: 30})
type Record struct {
	Fields []RecordField
	Line   uint
}

type RecordField struct {
	Name  string
	Value Expr
}

func (r *Record) Accept(visitor ExprVisitor) Value {
	return visitor.VisitRecord(r)
}

// EmptyRecord represents an empty record {}
type EmptyRecord struct {
	Line uint
}

func (e *EmptyRecord) Accept(visitor ExprVisitor) Value {
	return visitor.VisitEmptyRecord(e)
}

// List represents a list [1, 2, 3]
type List struct {
	Elements []Expr
	Line     uint
}

func (l *List) Accept(visitor ExprVisitor) Value {
	return visitor.VisitList(l)
}

// Access represents record field access (e.g., alice.name)
type Access struct {
	Object Expr
	Name   string
	Line   uint
}

func (a *Access) Accept(visitor ExprVisitor) Value {
	return visitor.VisitAccess(a)
}

// Builtin represents a builtin function call (e.g., !int_add(1, 2))
type Builtin struct {
	Name      string
	Arguments []Expr
	Line      uint
}

func (b *Builtin) Accept(visitor ExprVisitor) Value {
	return visitor.VisitBuiltin(b)
}

// Union represents a union type constructor (e.g., Cat("felix"))
type Union struct {
	Constructor string
	Value       Expr
	Line        uint
}

func (u *Union) Accept(visitor ExprVisitor) Value {
	return visitor.VisitUnion(u)
}

// Lambda represents a lambda expression (e.g., |x, y| { x + y })
type Lambda struct {
	Parameters []string
	Body       Expr
	Line       uint
}

func (l *Lambda) Accept(visitor ExprVisitor) Value {
	return visitor.VisitLambda(l)
}

// Match represents a match expression
type Match struct {
	Value Expr
	Cases []MatchCase
	Line  uint
}

type MatchCase struct {
	Pattern Expr
	Body    Expr
}

func (m *Match) Accept(visitor ExprVisitor) Value {
	return visitor.VisitMatch(m)
}

// Perform represents an effect (e.g., perform Log("hello"))
type Perform struct {
	Effect    string
	Arguments []Expr
	Line      uint
}

func (p *Perform) Accept(visitor ExprVisitor) Value {
	return visitor.VisitPerform(p)
}

// Handle represents a handle expression
type Handle struct {
	Effect   string
	Handler  Expr
	Fallback Expr
	Line     uint
}

func (h *Handle) Accept(visitor ExprVisitor) Value {
	return visitor.VisitHandle(h)
}

// NamedRef represents a named reference (e.g., @std:1)
type NamedRef struct {
	Module string
	Index  int
	Line   uint
}

func (n *NamedRef) Accept(visitor ExprVisitor) Value {
	return visitor.VisitNamedRef(n)
}

// Thunk represents a thunk (e.g., || {})
type Thunk struct {
	Body Expr
	Line uint
}

func (t *Thunk) Accept(visitor ExprVisitor) Value {
	return visitor.VisitThunk(t)
}

// Spread represents a spread operator (e.g., ..items)
type Spread struct {
	Expression Expr
	Line       uint
}

func (s *Spread) Accept(visitor ExprVisitor) Value {
	return visitor.VisitSpread(s)
}

// Destructure represents destructuring assignment
type Destructure struct {
	Fields []RecordField
	Line   uint
}

func (d *Destructure) Accept(visitor ExprVisitor) Value {
	return visitor.VisitDestructure(d)
}

// Seq represents a sequence of expressions
type Seq struct {
	Left  Expr
	Right Expr
	Line  uint
}

func (s *Seq) Accept(visitor ExprVisitor) Value {
	return visitor.VisitSeq(s)
}
