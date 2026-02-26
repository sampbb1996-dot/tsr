package kernel

import (
	"fmt"
	"strings"
)

type KTokenType int

const (
	KT_IDENT  KTokenType = iota
	KT_NUMBER
	KT_STRING
	KT_TRUE
	KT_FALSE
	KT_NIL
	KT_LBRACE
	KT_RBRACE
	KT_LPAREN
	KT_RPAREN
	KT_LBRACKET
	KT_RBRACKET
	KT_COMMA
	KT_COLON
	KT_ASSIGN
	KT_ARROW   // ->
	KT_PLUS
	KT_MINUS
	KT_STAR
	KT_SLASH
	KT_BANG
	KT_LT
	KT_GT
	KT_LE
	KT_GE
	KT_EQEQ
	KT_NEQ
	KT_AND
	KT_OR
	KT_NEWLINE
	KT_EOF
	KT_ILLEGAL
)

var kernelKeywords = map[string]KTokenType{
	"true":      KT_TRUE,
	"false":     KT_FALSE,
	"nil":       KT_NIL,
	"and":       KT_AND,
	"or":        KT_OR,
}

type KToken struct {
	Type   KTokenType
	Lexeme string
	Line   int
	Col    int
}

type KLexer struct {
	source []rune
	file   string
	pos    int
	line   int
	col    int
}

func NewKLexer(source, file string) *KLexer {
	return &KLexer{source: []rune(source), file: file, line: 1, col: 1}
}

func (l *KLexer) Tokenize() ([]KToken, error) {
	var tokens []KToken
	for l.pos < len(l.source) {
		tok, err := l.nextToken()
		if err != nil {
			return nil, err
		}
		if tok != nil {
			tokens = append(tokens, *tok)
		}
	}
	tokens = append(tokens, KToken{KT_EOF, "", l.line, l.col})
	return tokens, nil
}

func (l *KLexer) peek() rune {
	if l.pos >= len(l.source) {
		return 0
	}
	return l.source[l.pos]
}

func (l *KLexer) peekNext() rune {
	if l.pos+1 >= len(l.source) {
		return 0
	}
	return l.source[l.pos+1]
}

func (l *KLexer) advance() rune {
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

func (l *KLexer) nextToken() (*KToken, error) {
	startLine := l.line
	startCol := l.col
	ch := l.advance()

	switch {
	case ch == '#':
		for l.pos < len(l.source) && l.peek() != '\n' {
			l.advance()
		}
		return nil, nil
	case ch == '\n':
		return &KToken{KT_NEWLINE, "\n", startLine, startCol}, nil
	case ch == ' ' || ch == '\t' || ch == '\r':
		return nil, nil
	case ch == '{':
		return &KToken{KT_LBRACE, "{", startLine, startCol}, nil
	case ch == '}':
		return &KToken{KT_RBRACE, "}", startLine, startCol}, nil
	case ch == '(':
		return &KToken{KT_LPAREN, "(", startLine, startCol}, nil
	case ch == ')':
		return &KToken{KT_RPAREN, ")", startLine, startCol}, nil
	case ch == '[':
		return &KToken{KT_LBRACKET, "[", startLine, startCol}, nil
	case ch == ']':
		return &KToken{KT_RBRACKET, "]", startLine, startCol}, nil
	case ch == ',':
		return &KToken{KT_COMMA, ",", startLine, startCol}, nil
	case ch == ':':
		return &KToken{KT_COLON, ":", startLine, startCol}, nil
	case ch == '=':
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			return &KToken{KT_EQEQ, "==", startLine, startCol}, nil
		}
		return &KToken{KT_ASSIGN, "=", startLine, startCol}, nil
	case ch == '-':
		if l.pos < len(l.source) && l.peek() == '>' {
			l.advance()
			return &KToken{KT_ARROW, "->", startLine, startCol}, nil
		}
		return &KToken{KT_MINUS, "-", startLine, startCol}, nil
	case ch == '+':
		return &KToken{KT_PLUS, "+", startLine, startCol}, nil
	case ch == '*':
		return &KToken{KT_STAR, "*", startLine, startCol}, nil
	case ch == '/':
		return &KToken{KT_SLASH, "/", startLine, startCol}, nil
	case ch == '!':
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			return &KToken{KT_NEQ, "!=", startLine, startCol}, nil
		}
		return &KToken{KT_BANG, "!", startLine, startCol}, nil
	case ch == '<':
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			return &KToken{KT_LE, "<=", startLine, startCol}, nil
		}
		return &KToken{KT_LT, "<", startLine, startCol}, nil
	case ch == '>':
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			return &KToken{KT_GE, ">=", startLine, startCol}, nil
		}
		return &KToken{KT_GT, ">", startLine, startCol}, nil
	case ch == '"':
		s, err := l.scanString(startLine, startCol)
		if err != nil {
			return nil, err
		}
		return &KToken{KT_STRING, s, startLine, startCol}, nil
	case isKDigit(ch):
		num := l.scanNumber(ch)
		return &KToken{KT_NUMBER, num, startLine, startCol}, nil
	case isKAlpha(ch):
		ident := l.scanIdent(ch)
		if tt, ok := kernelKeywords[ident]; ok {
			return &KToken{tt, ident, startLine, startCol}, nil
		}
		return &KToken{KT_IDENT, ident, startLine, startCol}, nil
	default:
		return nil, fmt.Errorf("%s:%d:%d: kernel: unexpected character %q", l.file, startLine, startCol, ch)
	}
}

func (l *KLexer) scanString(startLine, startCol int) (string, error) {
	var sb strings.Builder
	for l.pos < len(l.source) {
		ch := l.advance()
		if ch == '"' {
			return sb.String(), nil
		}
		if ch == '\\' {
			if l.pos >= len(l.source) {
				return "", fmt.Errorf("%s:%d:%d: unterminated escape", l.file, startLine, startCol)
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

func (l *KLexer) scanNumber(first rune) string {
	var sb strings.Builder
	sb.WriteRune(first)
	hasDot := false
	for l.pos < len(l.source) {
		ch := l.peek()
		if isKDigit(ch) {
			sb.WriteRune(l.advance())
		} else if ch == '.' && !hasDot && isKDigit(l.peekNext()) {
			hasDot = true
			sb.WriteRune(l.advance())
		} else {
			break
		}
	}
	return sb.String()
}

func (l *KLexer) scanIdent(first rune) string {
	var sb strings.Builder
	sb.WriteRune(first)
	for l.pos < len(l.source) {
		ch := l.peek()
		if isKAlpha(ch) || isKDigit(ch) {
			sb.WriteRune(l.advance())
		} else {
			break
		}
	}
	return sb.String()
}

func isKDigit(ch rune) bool { return ch >= '0' && ch <= '9' }
func isKAlpha(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}
