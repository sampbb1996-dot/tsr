package tsr

import (
	"fmt"
	"strings"
)

type Lexer struct {
	source  []rune
	file    string
	pos     int
	line    int
	col     int
	tokens  []Token
}

func NewLexer(source, file string) *Lexer {
	return &Lexer{
		source: []rune(source),
		file:   file,
		line:   1,
		col:    1,
	}
}

func (l *Lexer) Tokenize() ([]Token, error) {
	for l.pos < len(l.source) {
		if err := l.scanToken(); err != nil {
			return nil, err
		}
	}
	l.tokens = append(l.tokens, Token{Type: TOKEN_EOF, Line: l.line, Col: l.col})
	return l.tokens, nil
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.source) {
		return 0
	}
	return l.source[l.pos]
}

func (l *Lexer) peekNext() rune {
	if l.pos+1 >= len(l.source) {
		return 0
	}
	return l.source[l.pos+1]
}

func (l *Lexer) advance() rune {
	ch := l.source[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

func (l *Lexer) addToken(t TokenType, lexeme string) {
	l.tokens = append(l.tokens, Token{Type: t, Lexeme: lexeme, Line: l.line, Col: l.col - len([]rune(lexeme))})
}

func (l *Lexer) addTokenAt(t TokenType, lexeme string, line, col int) {
	l.tokens = append(l.tokens, Token{Type: t, Lexeme: lexeme, Line: line, Col: col})
}

func (l *Lexer) scanToken() error {
	startLine := l.line
	startCol := l.col
	ch := l.advance()

	switch ch {
	case ' ', '\t', '\r', '\n':
		return nil
	case '#':
		for l.pos < len(l.source) && l.peek() != '\n' {
			l.advance()
		}
		return nil
	case '+':
		l.tokens = append(l.tokens, Token{TOKEN_PLUS, "+", startLine, startCol})
	case '-':
		l.tokens = append(l.tokens, Token{TOKEN_MINUS, "-", startLine, startCol})
	case '*':
		l.tokens = append(l.tokens, Token{TOKEN_STAR, "*", startLine, startCol})
	case '/':
		l.tokens = append(l.tokens, Token{TOKEN_SLASH, "/", startLine, startCol})
	case '(':
		l.tokens = append(l.tokens, Token{TOKEN_LPAREN, "(", startLine, startCol})
	case ')':
		l.tokens = append(l.tokens, Token{TOKEN_RPAREN, ")", startLine, startCol})
	case '{':
		l.tokens = append(l.tokens, Token{TOKEN_LBRACE, "{", startLine, startCol})
	case '}':
		l.tokens = append(l.tokens, Token{TOKEN_RBRACE, "}", startLine, startCol})
	case '[':
		l.tokens = append(l.tokens, Token{TOKEN_LBRACKET, "[", startLine, startCol})
	case ']':
		l.tokens = append(l.tokens, Token{TOKEN_RBRACKET, "]", startLine, startCol})
	case ',':
		l.tokens = append(l.tokens, Token{TOKEN_COMMA, ",", startLine, startCol})
	case ';':
		l.tokens = append(l.tokens, Token{TOKEN_SEMICOLON, ";", startLine, startCol})
	case ':':
		l.tokens = append(l.tokens, Token{TOKEN_COLON, ":", startLine, startCol})
	case '.':
		l.tokens = append(l.tokens, Token{TOKEN_DOT, ".", startLine, startCol})
	case '!':
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, Token{TOKEN_NEQ, "!=", startLine, startCol})
		} else {
			l.tokens = append(l.tokens, Token{TOKEN_BANG, "!", startLine, startCol})
		}
	case '<':
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, Token{TOKEN_LE, "<=", startLine, startCol})
		} else {
			l.tokens = append(l.tokens, Token{TOKEN_LT, "<", startLine, startCol})
		}
	case '>':
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, Token{TOKEN_GE, ">=", startLine, startCol})
		} else {
			l.tokens = append(l.tokens, Token{TOKEN_GT, ">", startLine, startCol})
		}
	case '=':
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, Token{TOKEN_EQEQ, "==", startLine, startCol})
		} else {
			l.tokens = append(l.tokens, Token{TOKEN_ASSIGN, "=", startLine, startCol})
		}
	case '"':
		s, err := l.scanString(startLine, startCol)
		if err != nil {
			return err
		}
		l.tokens = append(l.tokens, Token{TOKEN_STRING, s, startLine, startCol})
	default:
		if isDigit(ch) {
			num := l.scanNumber(ch)
			l.tokens = append(l.tokens, Token{TOKEN_NUMBER, num, startLine, startCol})
		} else if isAlpha(ch) {
			ident := l.scanIdent(ch)
			tt, ok := keywords[ident]
			if !ok {
				tt = TOKEN_IDENT
			}
			l.tokens = append(l.tokens, Token{tt, ident, startLine, startCol})
		} else {
			return fmt.Errorf("%s:%d:%d: unexpected character %q", l.file, startLine, startCol, ch)
		}
	}
	return nil
}

func (l *Lexer) scanString(startLine, startCol int) (string, error) {
	var sb strings.Builder
	for l.pos < len(l.source) {
		ch := l.advance()
		if ch == '"' {
			return sb.String(), nil
		}
		if ch == '\\' {
			if l.pos >= len(l.source) {
				return "", fmt.Errorf("%s:%d:%d: unterminated escape in string", l.file, startLine, startCol)
			}
			esc := l.advance()
			switch esc {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case '\\':
				sb.WriteRune('\\')
			case '"':
				sb.WriteRune('"')
			default:
				return "", fmt.Errorf("%s:%d:%d: unknown escape \\%c", l.file, startLine, startCol, esc)
			}
		} else if ch == '\n' {
			return "", fmt.Errorf("%s:%d:%d: unterminated string", l.file, startLine, startCol)
		} else {
			sb.WriteRune(ch)
		}
	}
	return "", fmt.Errorf("%s:%d:%d: unterminated string", l.file, startLine, startCol)
}

func (l *Lexer) scanNumber(first rune) string {
	var sb strings.Builder
	sb.WriteRune(first)
	hasDot := false
	for l.pos < len(l.source) {
		ch := l.peek()
		if isDigit(ch) {
			sb.WriteRune(l.advance())
		} else if ch == '.' && !hasDot && isDigit(l.peekNext()) {
			hasDot = true
			sb.WriteRune(l.advance())
		} else {
			break
		}
	}
	return sb.String()
}

func (l *Lexer) scanIdent(first rune) string {
	var sb strings.Builder
	sb.WriteRune(first)
	for l.pos < len(l.source) {
		ch := l.peek()
		if isAlpha(ch) || isDigit(ch) {
			sb.WriteRune(l.advance())
		} else {
			break
		}
	}
	return sb.String()
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isAlpha(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}
