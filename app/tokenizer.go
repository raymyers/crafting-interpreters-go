package main

import (
	"bufio"
	"io"
	"os"
	"strings"
)

func TokenizeFile(filename string) ([]Token, error) {
	file, err := os.Open(filename)
	if err != nil {
		return make([]Token, 0), err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	return TokenizeReader(reader)
}

func TokenizeString(text string) ([]Token, error) {
	reader := strings.NewReader(text)
	return TokenizeReader(bufio.NewReader(reader))
}

func TokenizeReader(reader *bufio.Reader) ([]Token, error) {
	result := make([]Token, 0)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err != io.EOF {
				return result, err
			}

			result = append(result, Token{EOF, "", ""})
			break
		}

		switch b {
		case '(':
			result = append(result, Token{LPAR, "(", ""})
		case ')':
			result = append(result, Token{RPAR, ")", ""})
		case '{':
			result = append(result, Token{LBRAC, "{", ""})
		case '}':
			result = append(result, Token{RBRAC, "}", ""})
		case '*':
			result = append(result, Token{STAR, "*", ""})
		case '.':
			result = append(result, Token{DOT, ".", ""})
		case ',':
			result = append(result, Token{COMMA, ",", ""})
		case '+':
			result = append(result, Token{PLUS, "+", ""})
		case '-':
			result = append(result, Token{MINUS, "-", ""})
		case ';':
			result = append(result, Token{SEMICOLON, ";", ""})
		}

	}

	return result, nil
}
