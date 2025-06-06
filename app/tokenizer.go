package main

import (
	"bufio"
	"fmt"
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
	var errors []string
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
		case '!':
			next, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return result, err
				}
				result = append(result, Token{BANG, "!", ""})
				break
			}
			if next == '=' {
				result = append(result, Token{BANG_EQUAL, "!=", ""})
			} else {
				reader.UnreadByte()
				result = append(result, Token{BANG, "!", ""})
			}
		case '=':
			next, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return result, err
				}
				result = append(result, Token{EQUAL, "=", ""})
				break
			}
			if next == '=' {
				result = append(result, Token{EQUAL_EQUAL, "==", ""})
			} else {
				reader.UnreadByte()
				result = append(result, Token{EQUAL, "=", ""})
			}
		case '<':
			next, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return result, err
				}
				result = append(result, Token{LESS, "<", ""})
				break
			}
			if next == '=' {
				result = append(result, Token{LESS_EQUAL, "<=", ""})
			} else {
				reader.UnreadByte()
				result = append(result, Token{LESS, "<", ""})
			}
		case '>':
			next, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return result, err
				}
				result = append(result, Token{GREATER, ">", ""})
				break
			}
			if next == '=' {
				result = append(result, Token{GREATER_EQUAL, ">=", ""})
			} else {
				reader.UnreadByte()
				result = append(result, Token{GREATER, ">", ""})
			}
		case '/':
			next, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return result, err
				}
				result = append(result, Token{SLASH, "/", ""})
				break
			}
			if next == '/' {
				_, err := reader.ReadString('\n')
				if err != nil && err != io.EOF {
					return result, err

				}
			} else {
				err := reader.UnreadByte()
				if err != nil {
					return nil, err
				}
				result = append(result, Token{SLASH, "/", ""})
			}
		case ' ':
			// Skip
		case '\t':
			// Skip
		default:
			_, err := fmt.Fprintf(os.Stderr, "[line 1] Error: Unexpected character: %c\n", b)
			if err != nil {
				return result, err
			}
			errors = append(errors, fmt.Sprintf("unexpected character: %c", b))
		}

	}
	if len(errors) > 0 {
		return result, fmt.Errorf("tokenization errors: %s", strings.Join(errors, "; "))
	}
	return result, nil
}
