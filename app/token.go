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
	STAR
	DOT
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
	STRING
	NUMBER
	IDENTIFIER
)

var tokenTypeName = map[TokenType]string{
	EOF:           "EOF",
	LPAR:          "LEFT_PAREN",
	RPAR:          "RIGHT_PAREN",
	LBRAC:         "LEFT_BRACE",
	RBRAC:         "RIGHT_BRACE",
	STAR:          "STAR",
	DOT:           "DOT",
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
	STRING:        "STRING",
	NUMBER:        "NUMBER",
	IDENTIFIER:    "IDENTIFIER",
}

type Token struct {
	Type    TokenType
	Lexeme  string
	Literal string
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
