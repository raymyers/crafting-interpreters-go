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
	VisitPrintStatement(expr *PrintStatement) Value
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

// PrintStatement (e.g., (1 + 2))
type PrintStatement struct {
	Expression Expr
	Line       uint
}

func (g *PrintStatement) Accept(visitor ExprVisitor) Value {
	return visitor.VisitPrintStatement(g)
}
