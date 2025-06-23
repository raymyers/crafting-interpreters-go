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
