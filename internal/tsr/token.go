package tsr

type TokenType int

const (
        // Literals
        TOKEN_NUMBER TokenType = iota
        TOKEN_STRING
        TOKEN_IDENT
        TOKEN_TRUE
        TOKEN_FALSE
        TOKEN_NIL

        // Keywords
        TOKEN_LET
        TOKEN_FN
        TOKEN_RETURN
        TOKEN_IF
        TOKEN_ELSE
        TOKEN_WHILE
        TOKEN_REGIME
        TOKEN_STRAIN
        TOKEN_COMMIT
        TOKEN_CAPABILITY
        TOKEN_IMPORT
        TOKEN_AS
        TOKEN_AND
        TOKEN_OR
        TOKEN_SAY

        // Operators
        TOKEN_PLUS
        TOKEN_MINUS
        TOKEN_STAR
        TOKEN_SLASH
        TOKEN_BANG
        TOKEN_LT
        TOKEN_GT
        TOKEN_LE
        TOKEN_GE
        TOKEN_EQEQ
        TOKEN_NEQ
        TOKEN_ASSIGN

        // Punctuation
        TOKEN_COLON
        TOKEN_DOT
        TOKEN_COMMA
        TOKEN_SEMICOLON
        TOKEN_LPAREN
        TOKEN_RPAREN
        TOKEN_LBRACE
        TOKEN_RBRACE
        TOKEN_LBRACKET
        TOKEN_RBRACKET

        TOKEN_EOF
        TOKEN_ILLEGAL
)

var keywords = map[string]TokenType{
        "let":    TOKEN_LET,
        "fn":     TOKEN_FN,
        "return": TOKEN_RETURN,
        "if":     TOKEN_IF,
        "else":   TOKEN_ELSE,
        "while":  TOKEN_WHILE,
        "true":   TOKEN_TRUE,
        "false":  TOKEN_FALSE,
        "nil":    TOKEN_NIL,
        "regime":     TOKEN_REGIME,
        "strain":     TOKEN_STRAIN,
        "commit":     TOKEN_COMMIT,
        "capability": TOKEN_CAPABILITY,
        "import":     TOKEN_IMPORT,
        "as":     TOKEN_AS,
        "and":    TOKEN_AND,
        "or":     TOKEN_OR,
        "say":    TOKEN_SAY,
}

type Token struct {
        Type    TokenType
        Lexeme  string
        Line    int
        Col     int
}
