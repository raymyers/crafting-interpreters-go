package main

import (
	"fmt"
	"strconv"
)

// Parser converts tokens into an AST
type Parser struct {
	tokens  []Token
	current int
}

// NewParser creates a new parser with the given tokens
func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens:  tokens,
		current: 0,
	}
}

// Parse parses the tokens into an expression
func (p *Parser) Parse() (Expr, error) {
	return p.statements()
}

// expression → equality
func (p *Parser) expression() (Expr, error) {
	return p.equality()
}

// equality → comparison ( ( "!=" | "==" ) comparison )*
func (p *Parser) equality() (Expr, error) {
	expr, err := p.comparison()
	if err != nil {
		return nil, err
	}

	for p.match(BANG_EQUAL, EQUAL_EQUAL) {
		operator := p.previous()
		right, err := p.comparison()
		if err != nil {
			return nil, err
		}
		expr = &Binary{Left: expr, Operator: operator, Right: right, Line: operator.Line}
	}

	return expr, nil
}

// comparison → term ( ( ">" | ">=" | "<" | "<=" ) term )*
func (p *Parser) comparison() (Expr, error) {
	expr, err := p.term()
	if err != nil {
		return nil, err
	}

	for p.match(GREATER, GREATER_EQUAL, LESS, LESS_EQUAL) {
		operator := p.previous()
		right, err := p.term()
		if err != nil {
			return nil, err
		}
		expr = &Binary{Left: expr, Operator: operator, Right: right, Line: operator.Line}
	}

	return expr, nil
}

// term → factor ( ( "-" | "+" ) factor )*
func (p *Parser) term() (Expr, error) {
	expr, err := p.factor()
	if err != nil {
		return nil, err
	}

	for p.match(MINUS, PLUS) {
		operator := p.previous()
		right, err := p.factor()
		if err != nil {
			return nil, err
		}
		expr = &Binary{Left: expr, Operator: operator, Right: right, Line: operator.Line}
	}

	return expr, nil
}

// factor → unary ( ( "/" | "*" ) unary )*
func (p *Parser) factor() (Expr, error) {
	expr, err := p.unary()
	if err != nil {
		return nil, err
	}

	for p.match(SLASH, STAR) {
		operator := p.previous()
		right, err := p.unary()
		if err != nil {
			return nil, err
		}
		expr = &Binary{Left: expr, Operator: operator, Right: right, Line: operator.Line}
	}

	return expr, nil
}

// unary → ( "!" | "-" ) unary | primary
func (p *Parser) unary() (Expr, error) {
	if p.match(BANG, MINUS) {
		operator := p.previous()
		right, err := p.unary()
		if err != nil {
			return nil, err
		}
		return &Unary{Operator: operator, Right: right, Line: operator.Line}, nil
	}

	return p.primary()
}

// statements → expression (";" expression)* | ";"
func (p *Parser) statements() (Expr, error) {
	var results []Expr
	expr, err := p.expression()
	if err != nil {
		return nil, err
	}
	line := p.previous().Line
	results = append(results, expr)
	for {
		if !p.match(SEMICOLON) {
			break
		}
		expr, err := p.expression()

		if err != nil {
			break
		}
		results = append(results, expr)
	}

	if len(results) == 1 {
		return results[0], nil
	}
	return &Statements{Exprs: results, Line: line}, nil

}

// primary → NUMBER | STRING | "true" | "false" | "nil"
//
//	| "(" expression ")" | printStatement | varStatement
func (p *Parser) primary() (Expr, error) {
	if p.match(FALSE) {
		return &Literal{Value: BoolValue{Val: false}, Line: p.previous().Line}, nil
	}

	if p.match(TRUE) {
		return &Literal{Value: BoolValue{Val: true}, Line: p.previous().Line}, nil
	}

	if p.match(NIL) {
		return &Literal{Value: NilValue{}, Line: p.previous().Line}, nil
	}

	if p.match(NUMBER) {
		token := p.previous()
		value, err := strconv.ParseFloat(token.Lexeme, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number: %s", token.Lexeme)
		}
		return &Literal{Value: NumberValue{Val: value}, Line: token.Line}, nil
	}

	if p.match(STRING) {
		token := p.previous()
		// Remove quotes from string literal
		value := token.Literal
		return &Literal{Value: StringValue{Val: value}, Line: token.Line}, nil
	}

	if p.match(LPAR) {
		expr, err := p.expression()
		if err != nil {
			return nil, err
		}
		_, err = p.consume(RPAR, "Expect ')' after expression.")
		if err != nil {
			return nil, err
		}
		return &Grouping{Expression: expr, Line: p.tokens[p.current-2].Line}, nil
	}

	if p.match(PRINT) {
		expr, err := p.expression()
		if err != nil {
			return nil, err
		}

		return &PrintStatement{Expression: expr, Line: p.tokens[p.current-2].Line}, nil
	}
	if p.match(VAR) {
		if !p.match(IDENTIFIER) {
			return nil, fmt.Errorf("expect identifier")
		}
		varName := p.previous().Lexeme
		if !p.match(EQUAL) {
			return &VarStatement{name: varName, Expression: &Literal{Value: NilValue{}, Line: p.previous().Line}, Line: p.tokens[p.current-2].Line}, nil
		}
		expr, err := p.expression()
		if err != nil {
			return nil, err
		}

		return &VarStatement{name: varName, Expression: expr, Line: p.tokens[p.current-2].Line}, nil
	}

	if p.match(IDENTIFIER) {
		token := p.previous()
		return &Variable{Name: token, Line: token.Line}, nil
	}

	return nil, fmt.Errorf("expect expression")
}

// Helper methods

func (p *Parser) match(types ...TokenType) bool {
	for _, tokenType := range types {
		if p.check(tokenType) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *Parser) check(tokenType TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Type == tokenType
}

func (p *Parser) advance() Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.previous()
}

func (p *Parser) isAtEnd() bool {
	return p.peek().Type == EOF
}

func (p *Parser) peek() Token {
	return p.tokens[p.current]
}

func (p *Parser) previous() Token {
	return p.tokens[p.current-1]
}

func (p *Parser) consume(tokenType TokenType, message string) (Token, error) {
	if p.check(tokenType) {
		return p.advance(), nil
	}
	return Token{}, fmt.Errorf("%s", message)
}
