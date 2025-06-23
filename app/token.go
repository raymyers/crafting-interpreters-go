package main

import (
	"fmt"
)

type TokenType int

const (
	EOF TokenType = iota
	LPAR
	RPAR
	LBRAC
	RBRAC
	LEFT_BRACKET
	RIGHT_BRACKET
	STAR
	DOT
	DOT_DOT
	COMMA
	PLUS
	MINUS
	SEMICOLON
	BANG
	BANG_EQUAL
	EQUAL
	EQUAL_EQUAL
	LESS
	LESS_EQUAL
	GREATER
	GREATER_EQUAL
	SLASH
	PIPE
	PIPE_PIPE
	AT
	COLON
	ARROW
	HASH
	STRING
	NUMBER
	IDENTIFIER
	AND
	ELSE
	IF
	NIL
	OR
	MATCH
	PERFORM
	HANDLE
	NOT
	UNDERSCORE
)

var tokenTypeName = map[TokenType]string{
	EOF:           "EOF",
	LPAR:          "LEFT_PAREN",
	RPAR:          "RIGHT_PAREN",
	LBRAC:         "LEFT_BRACE",
	RBRAC:         "RIGHT_BRACE",
	LEFT_BRACKET:  "LEFT_BRACKET",
	RIGHT_BRACKET: "RIGHT_BRACKET",
	STAR:          "STAR",
	DOT:           "DOT",
	DOT_DOT:       "DOT_DOT",
	COMMA:         "COMMA",
	PLUS:          "PLUS",
	MINUS:         "MINUS",
	SEMICOLON:     "SEMICOLON",
	BANG:          "BANG",
	BANG_EQUAL:    "BANG_EQUAL",
	EQUAL:         "EQUAL",
	EQUAL_EQUAL:   "EQUAL_EQUAL",
	LESS:          "LESS",
	LESS_EQUAL:    "LESS_EQUAL",
	GREATER:       "GREATER",
	GREATER_EQUAL: "GREATER_EQUAL",
	SLASH:         "SLASH",
	PIPE:          "PIPE",
	PIPE_PIPE:     "PIPE_PIPE",
	AT:            "AT",
	COLON:         "COLON",
	ARROW:         "ARROW",
	HASH:          "HASH",
	STRING:        "STRING",
	NUMBER:        "NUMBER",
	IDENTIFIER:    "IDENTIFIER",
	AND:           "AND",
	ELSE:          "ELSE",
	IF:            "IF",
	NIL:           "NIL",
	OR:            "OR",
	MATCH:         "MATCH",
	PERFORM:       "PERFORM",
	HANDLE:        "HANDLE",
	NOT:           "NOT",
	UNDERSCORE:    "UNDERSCORE",
}

type Token struct {
	Type    TokenType
	Lexeme  string
	Literal string
	Line    uint
}

func (t *Token) String() string {
	s := fmt.Sprintf("%v %s ", tokenTypeName[t.Type], t.Lexeme)

	if t.Literal != "" {
		s += t.Literal
	} else {
		s += "null"
	}

	return s
}
