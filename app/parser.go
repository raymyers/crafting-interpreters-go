package main

import (
	"fmt"
	"strconv"
	"strings"
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

// expression → assignment
func (p *Parser) expression() (Expr, error) {
	return p.assignment()
}

// assignment → equality ( "=" assignment )*
func (p *Parser) assignment() (Expr, error) {
	expr, err := p.equality()
	if err != nil {
		return nil, err
	}

	if p.match(EQUAL) {
		operator := p.previous()
		right, err := p.assignment() // Right-associative
		if err != nil {
			return nil, err
		}
		
		// Check if left side is a record pattern for destructuring
		if record, ok := expr.(*Record); ok {
			// Convert record to destructure pattern
			destructure := &Destructure{Fields: record.Fields, Line: record.Line}
			return &Binary{Left: destructure, Operator: operator, Right: right, Line: operator.Line}, nil
		}
		
		return &Binary{Left: expr, Operator: operator, Right: right, Line: operator.Line}, nil
	}

	return expr, nil
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

// comparison → term ( ( "or" | "and" | ">" | ">=" | "<" | "<=" ) term )*
func (p *Parser) comparison() (Expr, error) {
	expr, err := p.term()
	if err != nil {
		return nil, err
	}

	for p.match(OR, AND, GREATER, GREATER_EQUAL, LESS, LESS_EQUAL) {
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

// unary → ( "!" | "-" ) unary | "||" expression | call
func (p *Parser) unary() (Expr, error) {
	if p.match(PIPE_PIPE) {
		line := p.previous().Line
		body, err := p.expression()
		if err != nil {
			return nil, err
		}
		return &Thunk{Body: body, Line: line}, nil
	}
	if p.match(BANG) {
		operator := p.previous()
		// Check if this is a builtin call (!identifier(...))
		if p.check(IDENTIFIER) {
			name := p.advance().Lexeme
			if p.match(LPAR) {
				// Check if this looks like a builtin (lowercase identifier)
				if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' {
					// This is a builtin call
					var arguments []Expr
					if !p.check(RPAR) {
						for {
							arg, err := p.expression()
							if err != nil {
								return nil, err
							}
							arguments = append(arguments, arg)
							if !p.match(COMMA) {
								break
							}
						}
					}
					_, err := p.consume(RPAR, "Expect ')' after builtin arguments.")
					if err != nil {
						return nil, err
					}
					return &Builtin{Name: name, Arguments: arguments, Line: operator.Line}, nil
				} else {
					// Not a builtin call (uppercase identifier), treat as unary ! followed by call
					p.current-- // back up to re-parse the (
					p.current-- // back up to re-parse the identifier
					right, err := p.unary()
					if err != nil {
						return nil, err
					}
					return &Unary{Operator: operator, Right: right, Line: operator.Line}, nil
				}
			} else {
				// Not a builtin call, treat as unary ! followed by identifier
				p.current-- // back up to re-parse the identifier
				right, err := p.unary()
				if err != nil {
					return nil, err
				}
				return &Unary{Operator: operator, Right: right, Line: operator.Line}, nil
			}
		} else {
			right, err := p.unary()
			if err != nil {
				return nil, err
			}
			return &Unary{Operator: operator, Right: right, Line: operator.Line}, nil
		}
	}
	
	if p.match(MINUS) {
		operator := p.previous()
		right, err := p.unary()
		if err != nil {
			return nil, err
		}
		return &Unary{Operator: operator, Right: right, Line: operator.Line}, nil
	}

	return p.call()
}

// call → primary ( "(" arguments? ")" | "." IDENTIFIER )*
func (p *Parser) call() (Expr, error) {
	expr, err := p.primary()
	if err != nil {
		return nil, err
	}

	for {
		if p.match(LPAR) {
			expr, err = p.finishCall(expr)
			if err != nil {
				return nil, err
			}
		} else if p.match(DOT) {
			name, err := p.consume(IDENTIFIER, "Expect property name after '.'.")
			if err != nil {
				return nil, err
			}
			expr = &Access{Object: expr, Name: name.Lexeme, Line: name.Line}
		} else {
			break
		}
	}

	return expr, nil
}

// finishCall parses the arguments and creates a Call expression
func (p *Parser) finishCall(callee Expr) (Expr, error) {
	var arguments []Expr

	if !p.check(RPAR) {
		for {
			arg, err := p.expression()
			if err != nil {
				return nil, err
			}
			arguments = append(arguments, arg)

			if !p.match(COMMA) {
				break
			}
		}
	}

	paren, err := p.consume(RPAR, "Expect ')' after arguments.")
	if err != nil {
		return nil, err
	}

	// Check if this should be a union constructor
	if variable, ok := callee.(*Variable); ok {
		name := variable.Name.Lexeme
		// Check if this looks like a union constructor (starts with uppercase)
		if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
			// If there's exactly one argument, treat as union constructor
			if len(arguments) == 1 {
				return &Union{Constructor: name, Value: arguments[0], Line: paren.Line}, nil
			}
			// If there are no arguments, treat as union with empty record
			if len(arguments) == 0 {
				return &Union{Constructor: name, Value: &EmptyRecord{Line: paren.Line}, Line: paren.Line}, nil
			}
		}
	}

	return &Call{
		Callee:    callee,
		Arguments: arguments,
		Line:      paren.Line,
	}, nil
}

// statements → expression (";"? expression)* | ";"
// ; not required when Block is next
func (p *Parser) statements() (Expr, error) {
	var results []Expr
	expr, err := p.expression()
	if err != nil {
		return nil, err
	}
	line := p.previous().Line
	results = append(results, expr)
	for {
		_ = p.match(SEMICOLON)
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
//		| "(" expression ")" | printStatement | varStatement
//		| blockStatement | ifStatement | whileStatement | forStatement
//	 | fun
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

	if p.match(IF) {
		return p.ifStatement()
	}

	if p.match(WHILE) {
		return p.whileStatement()
	}
	if p.match(FOR) {
		return p.forStatement()
	}

	if p.match(IDENTIFIER) {
		token := p.previous()
		return &Variable{Name: token, Line: token.Line}, nil
	}

	if p.match(LBRAC) {
		return p.recordOrBlock()
	}
	
	if p.match(LEFT_BRACKET) {
		return p.listExpression()
	}
	
	if p.match(PIPE) {
		return p.lambda()
	}
	
	if p.match(AT) {
		return p.namedRef()
	}
	
	if p.match(PERFORM) {
		return p.performExpression()
	}
	
	if p.match(MATCH) {
		return p.matchExpression()
	}
	
	if p.match(HANDLE) {
		return p.handleExpression()
	}
	
	if p.match(FUN) {
		return p.funStatement()
	}
	return nil, fmt.Errorf("expect expression")
}

// blockStatement → "{" statements "}"
func (p *Parser) blockStatement() (Expr, error) {
	line := p.previous().Line
	var statements []Expr

	for !p.check(RBRAC) && !p.isAtEnd() {
		stmt, err := p.expression()
		if err != nil {
			return nil, err
		}
		statements = append(statements, stmt)

		// Optional semicolon after each statement
		p.match(SEMICOLON)
	}

	_, err := p.consume(RBRAC, "Expect '}' after block.")
	if err != nil {
		return nil, err
	}

	return &Block{Statements: statements, Line: line}, nil
}

// funStatement → "fun" ident "(" (ident ("," ident)*)? ")" block
func (p *Parser) funStatement() (Expr, error) {
	line := p.previous().Line
	var params []string
	name, err := p.consume(IDENTIFIER, "expect identifier after fun")
	if err != nil {
		return nil, err
	}
	_, err = p.consume(LPAR, "expect ( after function name")
	if err != nil {
		return nil, err
	}
	for !p.check(RPAR) {
		paramName, err := p.consume(IDENTIFIER, "expect arg name or )")
		if err != nil {
			return nil, err
		}

		params = append(params, paramName.Lexeme)
		if p.check(COMMA) {
			p.advance()
		} else {
			break
		}
	}
	_, err = p.consume(RPAR, "expect ) after arg list")
	if err != nil {
		return nil, err
	}
	_, err = p.consume(LBRAC, "expect { after arg list")
	if err != nil {
		return nil, err
	}
	blockExpr, err := p.blockStatement()
	if err != nil {
		return nil, err
	}
	if block, ok := blockExpr.(*Block); ok && block != nil {
		return &Fun{Name: name.Lexeme, Parameters: params, Block: *block, Line: line}, nil
	}
	return nil, fmt.Errorf("function body much be a block")
}

// ifStatement → "if" "(" expression ")" expression ( "else" expression )?
func (p *Parser) ifStatement() (Expr, error) {
	line := p.previous().Line

	_, err := p.consume(LPAR, "Expect '(' after 'if'.")
	if err != nil {
		return nil, err
	}

	condition, err := p.expression()
	if err != nil {
		return nil, err
	}

	_, err = p.consume(RPAR, "Expect ')' after if condition.")
	if err != nil {
		return nil, err
	}

	thenBranch, err := p.expression()
	if err != nil {
		return nil, err
	}

	var elseBranch Expr
	_ = p.match(SEMICOLON)
	if p.match(ELSE) {
		elseBranch, err = p.expression()
		if err != nil {
			return nil, err
		}
	}

	return &IfStatement{
		Condition:  condition,
		ThenBranch: thenBranch,
		ElseBranch: elseBranch,
		Line:       line,
	}, nil
}

// whileStatement → "while" "(" expression ")" expression
func (p *Parser) whileStatement() (Expr, error) {
	line := p.previous().Line

	_, err := p.consume(LPAR, "Expect '(' after 'while'.")
	if err != nil {
		return nil, err
	}

	condition, err := p.expression()
	if err != nil {
		return nil, err
	}

	_, err = p.consume(RPAR, "Expect ')' after while condition.")
	if err != nil {
		return nil, err
	}

	body, err := p.expression()
	if err != nil {
		return nil, err
	}

	return &WhileStatement{
		Condition: condition,
		Body:      body,
		Line:      line,
	}, nil
}

// forStatement → "for" "(" expression ";" expression ";" expression ")" expression
func (p *Parser) forStatement() (Expr, error) {
	line := p.previous().Line

	_, err := p.consume(LPAR, "Expect '(' after 'for'.")
	if err != nil {
		return nil, err
	}
	if p.check(LBRAC) {
		return nil, fmt.Errorf("can't use block as for initializer")
	}
	// Optional
	initializer, _ := p.expression()

	_, err = p.consume(SEMICOLON, "Expect ';' after for initializer.")
	if err != nil {
		return nil, err
	}
	if p.check(LBRAC) {
		return nil, fmt.Errorf("can't use block as for condition")
	}
	// Optional
	condition, _ := p.expression()

	_, err = p.consume(SEMICOLON, "expect ';' after for condition.")
	if err != nil {
		return nil, err
	}
	if p.check(LBRAC) {
		return nil, fmt.Errorf("can't use block as for increment")
	}
	// Optional
	increment, _ := p.expression()

	_, err = p.consume(RPAR, "Expect ')' after for condition.")
	if err != nil {
		return nil, err
	}
	if p.check(VAR) {
		return nil, fmt.Errorf("can't declare var as single statement in for")
	}
	body, err := p.expression()
	if err != nil {
		return nil, err
	}

	return &ForStatement{
		Initializer: initializer,
		Condition:   condition,
		Increment:   increment,
		Body:        body,
		Line:        line,
	}, nil
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

// recordOrBlock determines if {} is an empty record or a block based on content
func (p *Parser) recordOrBlock() (Expr, error) {
	line := p.previous().Line
	
	// Check if it's empty {}
	if p.check(RBRAC) {
		p.advance() // consume }
		return &EmptyRecord{Line: line}, nil
	}
	
	// Look ahead to see if this looks like a record (has : after identifier)
	saved := p.current
	isRecord := false
	
	if p.check(IDENTIFIER) {
		p.advance()
		if p.check(COLON) {
			isRecord = true
		}
	}
	
	// Restore position
	p.current = saved
	
	if isRecord {
		return p.recordStatement()
	} else {
		return p.blockStatement()
	}
}

// recordStatement → "{" (identifier ":" expression ("," identifier ":" expression)*)? "}"
func (p *Parser) recordStatement() (Expr, error) {
	line := p.previous().Line
	var fields []RecordField
	
	for !p.check(RBRAC) && !p.isAtEnd() {
		name, err := p.consume(IDENTIFIER, "Expect field name.")
		if err != nil {
			return nil, err
		}
		
		_, err = p.consume(COLON, "Expect ':' after field name.")
		if err != nil {
			return nil, err
		}
		
		value, err := p.expression()
		if err != nil {
			return nil, err
		}
		
		fields = append(fields, RecordField{Name: name.Lexeme, Value: value})
		
		if !p.match(COMMA) {
			break
		}
	}
	
	_, err := p.consume(RBRAC, "Expect '}' after record.")
	if err != nil {
		return nil, err
	}
	
	return &Record{Fields: fields, Line: line}, nil
}

// listExpression → "[" (expression ("," expression)*)? "]"
func (p *Parser) listExpression() (Expr, error) {
	line := p.previous().Line
	var elements []Expr
	
	if !p.check(RIGHT_BRACKET) {
		for {
			// Check for spread operator
			if p.match(DOT_DOT) {
				expr, err := p.expression()
				if err != nil {
					return nil, err
				}
				elements = append(elements, &Spread{Expression: expr, Line: p.previous().Line})
			} else {
				expr, err := p.expression()
				if err != nil {
					return nil, err
				}
				elements = append(elements, expr)
			}
			
			if !p.match(COMMA) {
				break
			}
		}
	}
	
	_, err := p.consume(RIGHT_BRACKET, "Expect ']' after list elements.")
	if err != nil {
		return nil, err
	}
	
	return &List{Elements: elements, Line: line}, nil
}

// namedRef → "@" identifier ":" number
func (p *Parser) namedRef() (Expr, error) {
	line := p.previous().Line
	
	module, err := p.consume(IDENTIFIER, "Expect module name after '@'.")
	if err != nil {
		return nil, err
	}
	
	_, err = p.consume(COLON, "Expect ':' after module name.")
	if err != nil {
		return nil, err
	}
	
	indexToken, err := p.consume(NUMBER, "Expect number after ':'.")
	if err != nil {
		return nil, err
	}
	
	index, err := strconv.Atoi(indexToken.Lexeme)
	if err != nil {
		return nil, fmt.Errorf("invalid index: %s", indexToken.Lexeme)
	}
	
	return &NamedRef{Module: module.Lexeme, Index: index, Line: line}, nil
}

// lambda → "|" parameters "|" expression
func (p *Parser) lambda() (Expr, error) {
	line := p.previous().Line
	
	var parameters []string
	if !p.check(PIPE) {
		for {
			param, err := p.consume(IDENTIFIER, "Expect parameter name.")
			if err != nil {
				return nil, err
			}
			parameters = append(parameters, param.Lexeme)
			if !p.match(COMMA) {
				break
			}
		}
	}
	
	_, err := p.consume(PIPE, "Expect '|' after parameters.")
	if err != nil {
		return nil, err
	}
	
	body, err := p.expression()
	if err != nil {
		return nil, err
	}
	
	// If the body is a block with a single expression, unwrap it
	if block, ok := body.(*Block); ok && len(block.Statements) == 1 {
		if expr, ok := block.Statements[0].(Expr); ok {
			body = expr
		}
	}
	
	return &Lambda{Parameters: parameters, Body: body, Line: line}, nil
}

// performExpression → "perform" identifier "(" arguments ")"
func (p *Parser) performExpression() (Expr, error) {
	line := p.previous().Line
	
	effect, err := p.consume(IDENTIFIER, "Expect effect name after 'perform'.")
	if err != nil {
		return nil, err
	}
	
	_, err = p.consume(LPAR, "Expect '(' after effect name.")
	if err != nil {
		return nil, err
	}
	
	var arguments []Expr
	if !p.check(RPAR) {
		for {
			arg, err := p.expression()
			if err != nil {
				return nil, err
			}
			arguments = append(arguments, arg)
			if !p.match(COMMA) {
				break
			}
		}
	}
	
	_, err = p.consume(RPAR, "Expect ')' after arguments.")
	if err != nil {
		return nil, err
	}
	
	return &Perform{Effect: effect.Lexeme, Arguments: arguments, Line: line}, nil
}

// matchExpression → "match" expression "{" matchCase* "}"
func (p *Parser) matchExpression() (Expr, error) {
	line := p.previous().Line
	
	value, err := p.expression()
	if err != nil {
		return nil, err
	}
	
	_, err = p.consume(LBRAC, "Expect '{' after match value.")
	if err != nil {
		return nil, err
	}
	
	var cases []MatchCase
	for !p.check(RBRAC) && !p.isAtEnd() {
		// Parse pattern: Constructor(params) or Constructor(_)
		constructor, err := p.consume(IDENTIFIER, "Expect constructor name.")
		if err != nil {
			return nil, err
		}
		
		_, err = p.consume(LPAR, "Expect '(' after constructor.")
		if err != nil {
			return nil, err
		}
		
		var params []string
		if !p.check(RPAR) {
			for {
				param, err := p.consume(IDENTIFIER, "Expect parameter name.")
				if err != nil {
					return nil, err
				}
				params = append(params, param.Lexeme)
				if !p.match(COMMA) {
					break
				}
			}
		}
		
		_, err = p.consume(RPAR, "Expect ')' after parameters.")
		if err != nil {
			return nil, err
		}
		
		_, err = p.consume(ARROW, "Expect '->' after pattern.")
		if err != nil {
			return nil, err
		}
		
		body, err := p.expression()
		if err != nil {
			return nil, err
		}
		
		// Create pattern expression
		pattern := &Union{Constructor: constructor.Lexeme, Value: &Variable{Name: Token{Lexeme: strings.Join(params, " ")}, Line: constructor.Line}, Line: constructor.Line}
		cases = append(cases, MatchCase{Pattern: pattern, Body: body})
	}
	
	_, err = p.consume(RBRAC, "Expect '}' after match cases.")
	if err != nil {
		return nil, err
	}
	
	return &Match{Value: value, Cases: cases, Line: line}, nil
}

// handleExpression → "handle" identifier "(" expression "," expression ")"
func (p *Parser) handleExpression() (Expr, error) {
	line := p.previous().Line
	
	effect, err := p.consume(IDENTIFIER, "Expect effect name after 'handle'.")
	if err != nil {
		return nil, err
	}
	
	_, err = p.consume(LPAR, "Expect '(' after effect name.")
	if err != nil {
		return nil, err
	}
	
	handler, err := p.expression()
	if err != nil {
		return nil, err
	}
	
	_, err = p.consume(COMMA, "Expect ',' after handler.")
	if err != nil {
		return nil, err
	}
	
	fallback, err := p.expression()
	if err != nil {
		return nil, err
	}
	
	_, err = p.consume(RPAR, "Expect ')' after fallback.")
	if err != nil {
		return nil, err
	}
	
	return &Handle{Effect: effect.Lexeme, Handler: handler, Fallback: fallback, Line: line}, nil
}
