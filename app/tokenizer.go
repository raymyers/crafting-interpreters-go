package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"
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
	var lineNo uint = 1
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err != io.EOF {
				return result, err
			}

			result = append(result, Token{EOF, "", "", lineNo})
			break
		}

		switch b {
		case '(':
			result = append(result, Token{LPAR, "(", "", lineNo})
		case ')':
			result = append(result, Token{RPAR, ")", "", lineNo})
		case '{':
			result = append(result, Token{LBRAC, "{", "", lineNo})
		case '}':
			result = append(result, Token{RBRAC, "}", "", lineNo})
		case '[':
			result = append(result, Token{LEFT_BRACKET, "[", "", lineNo})
		case ']':
			result = append(result, Token{RIGHT_BRACKET, "]", "", lineNo})
		case '*':
			result = append(result, Token{STAR, "*", "", lineNo})
		case '.':
			next, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return result, err
				}
				result = append(result, Token{DOT, ".", "", lineNo})
				break
			}
			if next == '.' {
				result = append(result, Token{DOT_DOT, "..", "", lineNo})
			} else {
				reader.UnreadByte()
				result = append(result, Token{DOT, ".", "", lineNo})
			}
		case ',':
			result = append(result, Token{COMMA, ",", "", lineNo})
		case '+':
			result = append(result, Token{PLUS, "+", "", lineNo})
		case '-':
			next, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return result, err
				}
				result = append(result, Token{MINUS, "-", "", lineNo})
				break
			}
			if next == '>' {
				result = append(result, Token{ARROW, "->", "", lineNo})
			} else {
				reader.UnreadByte()
				result = append(result, Token{MINUS, "-", "", lineNo})
			}
		case ';':
			result = append(result, Token{SEMICOLON, ";", "", lineNo})
		case '!':
			next, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return result, err
				}
				result = append(result, Token{BANG, "!", "", lineNo})
				break
			}
			if next == '=' {
				result = append(result, Token{BANG_EQUAL, "!=", "", lineNo})
			} else if unicode.IsLetter(rune(next)) && next >= 'a' && next <= 'z' {
				// This is a builtin function !identifier
				// Read the rest of the identifier
				idStr, _, err2 := readIdentifier(reader, next, result)
				if err2 != nil {
					return result, err2
				}
				// Create a special identifier token with ! prefix
				result = append(result, Token{IDENTIFIER, "!" + idStr, "", lineNo})
			} else {
				reader.UnreadByte()
				result = append(result, Token{BANG, "!", "", lineNo})
			}
		case '=':
			next, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return result, err
				}
				result = append(result, Token{EQUAL, "=", "", lineNo})
				break
			}
			if next == '=' {
				result = append(result, Token{EQUAL_EQUAL, "==", "", lineNo})
			} else {
				reader.UnreadByte()
				result = append(result, Token{EQUAL, "=", "", lineNo})
			}
		case '<':
			next, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return result, err
				}
				result = append(result, Token{LESS, "<", "", lineNo})
				break
			}
			if next == '=' {
				result = append(result, Token{LESS_EQUAL, "<=", "", lineNo})
			} else {
				reader.UnreadByte()
				result = append(result, Token{LESS, "<", "", lineNo})
			}
		case '>':
			next, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return result, err
				}
				result = append(result, Token{GREATER, ">", "", lineNo})
				break
			}
			if next == '=' {
				result = append(result, Token{GREATER_EQUAL, ">=", "", lineNo})
			} else {
				reader.UnreadByte()
				result = append(result, Token{GREATER, ">", "", lineNo})
			}
		case '/':
			next, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return result, err
				}
				result = append(result, Token{SLASH, "/", "", lineNo})
				break
			}
			if next == '/' {
				_, err := reader.ReadString('\n')
				if err != nil && err != io.EOF {
					return result, err

				}
				lineNo++
			} else {
				err := reader.UnreadByte()
				if err != nil {
					return nil, err
				}
				result = append(result, Token{SLASH, "/", "", lineNo})
			}
		case '|':
			next, err := reader.ReadByte()
			if err != nil {
				if err != io.EOF {
					return result, err
				}
				result = append(result, Token{PIPE, "|", "", lineNo})
				break
			}
			if next == '|' {
				result = append(result, Token{PIPE_PIPE, "||", "", lineNo})
			} else {
				reader.UnreadByte()
				result = append(result, Token{PIPE, "|", "", lineNo})
			}
		case '@':
			result = append(result, Token{AT, "@", "", lineNo})
		case ':':
			result = append(result, Token{COLON, ":", "", lineNo})
		case '#':
			// Hash comment - skip to end of line
			_, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				return result, err
			}
			lineNo++
		case ' ':
			// Skip
		case '\t':
			// Skip
		case '\n':
			lineNo++
		case '\r':
			// Skip
		case '"':
			// String literal
			var stringValue strings.Builder
			for {
				b, err := reader.ReadByte()
				if err != nil {
					if err == io.EOF {
						_, err := fmt.Fprintf(os.Stderr, "[line %d] Error: Unterminated string.\n", lineNo)
						if err != nil {
							return result, err
						}
						errors = append(errors, "unterminated string")
						break
					}
					return result, err
				}

				if b == '"' {
					// End of string
					result = append(result, Token{STRING, fmt.Sprintf("\"%s\"", stringValue.String()), stringValue.String(), lineNo})
					break
				} else if b == '\n' {
					lineNo++
					stringValue.WriteByte(b)
				} else {
					stringValue.WriteByte(b)
				}
			}
		default:
			if unicode.IsDigit(rune(b)) {
				numStr, tokens, err2 := readNumberLiteral(reader, b, result)
				if err2 != nil {
					return tokens, err2
				}
				// Parse as float to get the literal value
				floatVal, err := strconv.ParseFloat(numStr, 64)
				if err != nil {
					_, err := fmt.Fprintf(os.Stderr, "[line %d] Error: Invalid number: %s\n", lineNo, numStr)
					if err != nil {
						return result, err
					}
					errors = append(errors, fmt.Sprintf("invalid number: %s", numStr))
				} else {
					// Format with minimum 1 decimal place but only as many as needed
					formatted := fmt.Sprintf("%g", floatVal)
					// If no decimal point, add .0
					if !strings.Contains(formatted, ".") {
						formatted += ".0"
					}
					result = append(result, Token{NUMBER, numStr, formatted, lineNo})
				}
			} else if unicode.IsLetter(rune(b)) || b == '_' {
				idStr, tokens, err2 := readIdentifier(reader, b, result)
				if err2 != nil {
					return tokens, err2
				}

				if err != nil {
					_, err := fmt.Fprintf(os.Stderr, "[line %d] Error: Invalid number: %s\n", lineNo, idStr)
					if err != nil {
						return result, err
					}
					errors = append(errors, fmt.Sprintf("invalid number: %s", idStr))
				} else {
					// Check if identifier is a reserved word
					tokenType := getTokenTypeForIdentifier(idStr)
					result = append(result, Token{tokenType, idStr, "", lineNo})
				}
			} else {
				_, err := fmt.Fprintf(os.Stderr, "[line %d] Error: Unexpected character: %c\n", lineNo, b)
				if err != nil {
					return result, err
				}
				errors = append(errors, fmt.Sprintf("unexpected character: %c", b))
			}
		}

	}
	if len(errors) > 0 {
		return result, fmt.Errorf("tokenization errors: %s", strings.Join(errors, "; "))
	}
	return result, nil
}

func readNumberLiteral(reader *bufio.Reader, b byte, result []Token) (string, []Token, error) {
	// Number literal
	var numberStr strings.Builder
	numberStr.WriteByte(b)

	for {
		next, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", result, err
		}

		if unicode.IsDigit(rune(next)) || next == '.' {
			numberStr.WriteByte(next)
		} else {
			reader.UnreadByte()
			break
		}
	}

	numStr := numberStr.String()
	return numStr, nil, nil
}

func readIdentifier(reader *bufio.Reader, b byte, result []Token) (string, []Token, error) {
	var numberStr strings.Builder
	numberStr.WriteByte(b)

	for {
		next, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", result, err
		}

		if unicode.IsDigit(rune(next)) || unicode.IsLetter(rune(next)) || next == '_' {
			numberStr.WriteByte(next)
		} else {
			reader.UnreadByte()
			break
		}
	}

	numStr := numberStr.String()
	return numStr, nil, nil
}

func getTokenTypeForIdentifier(identifier string) TokenType {
	switch identifier {
	case "_":
		return UNDERSCORE
	case "and":
		return AND
	case "else":
		return ELSE
	case "if":
		return IF
	case "nil":
		return NIL
	case "or":
		return OR
	case "match":
		return MATCH
	case "perform":
		return PERFORM
	case "handle":
		return HANDLE
	case "not":
		return NOT
	default:
		return IDENTIFIER
	}
}
