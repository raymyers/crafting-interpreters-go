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
)

var tokenTypeName = map[TokenType]string{
	EOF:       "EOF",
	LPAR:      "LEFT_PAREN",
	RPAR:      "RIGHT_PAREN",
	LBRAC:     "LEFT_BRACE",
	RBRAC:     "RIGHT_BRACE",
	STAR:      "STAR",
	DOT:       "DOT",
	COMMA:     "COMMA",
	PLUS:      "PLUS",
	MINUS:     "MINUS",
	SEMICOLON: "SEMICOLON",
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
