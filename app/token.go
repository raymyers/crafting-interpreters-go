package main

import (
	"fmt"
)

type TokenType int

const (
	EOF TokenType = iota
	LPAR
	RPAR
)

var tokenTypeName = map[TokenType]string{
	EOF:  "EOF",
	LPAR: "LEFT_PAREN",
	RPAR: "RIGHT_PAREN",
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
