package tsr

import (
        "fmt"
        "math/big"
        "strconv"
)

type Parser struct {
        tokens []Token
        pos    int
        file   string
}

func NewParser(tokens []Token, file string) *Parser {
        return &Parser{tokens: tokens, file: file}
}

func (p *Parser) Parse() (*Program, error) {
        prog := &Program{}
        for !p.check(TOKEN_EOF) {
                stmt, err := p.parseStmt()
                if err != nil {
                        return nil, err
                }
                prog.Stmts = append(prog.Stmts, stmt)
        }
        return prog, nil
}

func (p *Parser) peek() Token {
        return p.tokens[p.pos]
}

func (p *Parser) check(t TokenType) bool {
        return p.tokens[p.pos].Type == t
}

func (p *Parser) advance() Token {
        tok := p.tokens[p.pos]
        if tok.Type != TOKEN_EOF {
                p.pos++
        }
        return tok
}

func (p *Parser) expect(t TokenType, context string) (Token, error) {
        if !p.check(t) {
                tok := p.peek()
                return Token{}, fmt.Errorf("%s:%d:%d: expected %s in %s, got %q", p.file, tok.Line, tok.Col, tokenTypeName(t), context, tok.Lexeme)
        }
        return p.advance(), nil
}

func (p *Parser) match(types ...TokenType) bool {
        for _, t := range types {
                if p.check(t) {
                        p.advance()
                        return true
                }
        }
        return false
}

func (p *Parser) parseStmt() (Stmt, error) {
        tok := p.peek()
        switch tok.Type {
        case TOKEN_LET:
                return p.parseLetStmt()
        case TOKEN_FN:
                return p.parseFnStmt()
        case TOKEN_RETURN:
                return p.parseReturnStmt()
        case TOKEN_IF:
                return p.parseIfStmt()
        case TOKEN_WHILE:
                return p.parseWhileStmt()
        case TOKEN_IMPORT:
                return p.parseImportStmt()
        case TOKEN_REGIME:
                return p.parseRegimeStmt()
        case TOKEN_STRAIN:
                return p.parseStrainStmt()
        case TOKEN_COMMIT:
                return p.parseCommitStmt()
        case TOKEN_CAPABILITY:
                return p.parseCapabilityStmt()
        case TOKEN_SAY:
                return p.parseSayStmt()
        case TOKEN_IDENT:
                // could be assignment or expression statement
                return p.parseIdentStmt()
        default:
                // expression statement
                return p.parseExprStmt()
        }
}

func (p *Parser) parseLetStmt() (Stmt, error) {
        tok := p.advance() // consume 'let'
        name, err := p.expect(TOKEN_IDENT, "let statement")
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(TOKEN_ASSIGN, "let statement"); err != nil {
                return nil, err
        }
        val, err := p.parseExpr()
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(TOKEN_SEMICOLON, "let statement"); err != nil {
                return nil, err
        }
        return &LetStmt{Line: tok.Line, Col: tok.Col, Name: name.Lexeme, Value: val}, nil
}

func (p *Parser) parseSayStmt() (Stmt, error) {
        tok := p.advance() // consume 'say'
        // say can be: say(expr); or just say expr;
        var val Expr
        var err error
        if p.check(TOKEN_LPAREN) {
                p.advance() // consume (
                val, err = p.parseExpr()
                if err != nil {
                        return nil, err
                }
                if _, err := p.expect(TOKEN_RPAREN, "say statement"); err != nil {
                        return nil, err
                }
        } else {
                val, err = p.parseExpr()
                if err != nil {
                        return nil, err
                }
        }
        if _, err := p.expect(TOKEN_SEMICOLON, "say statement"); err != nil {
                return nil, err
        }
        return &SayStmt{Line: tok.Line, Col: tok.Col, Value: val}, nil
}

func (p *Parser) parseIdentStmt() (Stmt, error) {
        // lookahead: if next is '=', it's assignment, else expression statement
        if len(p.tokens) > p.pos+1 && p.tokens[p.pos+1].Type == TOKEN_ASSIGN {
                return p.parseAssignStmt()
        }
        return p.parseExprStmt()
}

func (p *Parser) parseAssignStmt() (Stmt, error) {
        name := p.advance()
        p.advance() // consume '='
        val, err := p.parseExpr()
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(TOKEN_SEMICOLON, "assignment"); err != nil {
                return nil, err
        }
        return &AssignStmt{Line: name.Line, Col: name.Col, Name: name.Lexeme, Value: val}, nil
}

func (p *Parser) parseExprStmt() (Stmt, error) {
        tok := p.peek()
        expr, err := p.parseExpr()
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(TOKEN_SEMICOLON, "expression statement"); err != nil {
                return nil, err
        }
        return &ExprStmt{Line: tok.Line, Col: tok.Col, Expr: expr}, nil
}

func (p *Parser) parseFnStmt() (Stmt, error) {
        tok := p.advance() // consume 'fn'
        name, err := p.expect(TOKEN_IDENT, "fn declaration")
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(TOKEN_LPAREN, "fn params"); err != nil {
                return nil, err
        }
        var params []string
        for !p.check(TOKEN_RPAREN) && !p.check(TOKEN_EOF) {
                param, err := p.expect(TOKEN_IDENT, "fn param")
                if err != nil {
                        return nil, err
                }
                params = append(params, param.Lexeme)
                if !p.check(TOKEN_RPAREN) {
                        if _, err := p.expect(TOKEN_COMMA, "fn params"); err != nil {
                                return nil, err
                        }
                }
        }
        if _, err := p.expect(TOKEN_RPAREN, "fn params"); err != nil {
                return nil, err
        }
        body, err := p.parseBlock()
        if err != nil {
                return nil, err
        }
        return &FnStmt{Line: tok.Line, Col: tok.Col, Name: name.Lexeme, Params: params, Body: body}, nil
}

func (p *Parser) parseReturnStmt() (Stmt, error) {
        tok := p.advance() // consume 'return'
        var val Expr
        if !p.check(TOKEN_SEMICOLON) {
                var err error
                val, err = p.parseExpr()
                if err != nil {
                        return nil, err
                }
        }
        if _, err := p.expect(TOKEN_SEMICOLON, "return statement"); err != nil {
                return nil, err
        }
        if val == nil {
                val = &NilLit{Line: tok.Line, Col: tok.Col}
        }
        return &ReturnStmt{Line: tok.Line, Col: tok.Col, Value: val}, nil
}

func (p *Parser) parseIfStmt() (Stmt, error) {
        tok := p.advance() // consume 'if'
        if _, err := p.expect(TOKEN_LPAREN, "if condition"); err != nil {
                return nil, err
        }
        cond, err := p.parseExpr()
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(TOKEN_RPAREN, "if condition"); err != nil {
                return nil, err
        }
        consequence, err := p.parseBlock()
        if err != nil {
                return nil, err
        }
        var alternative []Stmt
        if p.check(TOKEN_ELSE) {
                p.advance()
                alternative, err = p.parseBlock()
                if err != nil {
                        return nil, err
                }
        }
        return &IfStmt{Line: tok.Line, Col: tok.Col, Condition: cond, Consequence: consequence, Alternative: alternative}, nil
}

func (p *Parser) parseWhileStmt() (Stmt, error) {
        tok := p.advance() // consume 'while'
        if _, err := p.expect(TOKEN_LPAREN, "while condition"); err != nil {
                return nil, err
        }
        cond, err := p.parseExpr()
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(TOKEN_RPAREN, "while condition"); err != nil {
                return nil, err
        }
        body, err := p.parseBlock()
        if err != nil {
                return nil, err
        }
        return &WhileStmt{Line: tok.Line, Col: tok.Col, Condition: cond, Body: body}, nil
}

func (p *Parser) parseImportStmt() (Stmt, error) {
        tok := p.advance() // consume 'import'
        path, err := p.expect(TOKEN_STRING, "import path")
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(TOKEN_AS, "import as"); err != nil {
                return nil, err
        }
        alias, err := p.expect(TOKEN_IDENT, "import alias")
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(TOKEN_SEMICOLON, "import statement"); err != nil {
                return nil, err
        }
        return &ImportStmt{Line: tok.Line, Col: tok.Col, Path: path.Lexeme, Alias: alias.Lexeme}, nil
}

func (p *Parser) parseRegimeStmt() (Stmt, error) {
        tok := p.advance() // consume 'regime'
        mode, err := p.expect(TOKEN_STRING, "regime mode")
        if err != nil {
                return nil, err
        }
        if mode.Lexeme != "CALM" && mode.Lexeme != "READONLY" {
                return nil, fmt.Errorf("%s:%d:%d: regime must be \"CALM\" or \"READONLY\", got %q", p.file, tok.Line, tok.Col, mode.Lexeme)
        }
        if _, err := p.expect(TOKEN_SEMICOLON, "regime statement"); err != nil {
                return nil, err
        }
        return &RegimeStmt{Line: tok.Line, Col: tok.Col, Mode: mode.Lexeme}, nil
}

func (p *Parser) parseCapabilityStmt() (Stmt, error) {
        tok := p.advance() // consume 'capability'
        kind, err := p.expect(TOKEN_IDENT, "capability kind")
        if err != nil {
                return nil, err
        }
        if kind.Lexeme != "file_write" && kind.Lexeme != "net" {
                return nil, fmt.Errorf("%s:%d:%d: capability kind must be \"file_write\" or \"net\", got %q",
                        p.file, tok.Line, tok.Col, kind.Lexeme)
        }
        pattern, err := p.expect(TOKEN_STRING, "capability pattern")
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(TOKEN_SEMICOLON, "capability statement"); err != nil {
                return nil, err
        }
        return &CapabilityStmt{Line: tok.Line, Col: tok.Col, Kind: kind.Lexeme, Pattern: pattern.Lexeme}, nil
}

func (p *Parser) parseStrainStmt() (Stmt, error) {
        tok := p.advance() // consume 'strain'
        if _, err := p.expect(TOKEN_LBRACE, "strain block"); err != nil {
                return nil, err
        }
        budget := defaultBudget()
        for !p.check(TOKEN_RBRACE) && !p.check(TOKEN_EOF) {
                key, err := p.expect(TOKEN_IDENT, "strain key")
                if err != nil {
                        return nil, err
                }
                if _, err := p.expect(TOKEN_COLON, "strain value"); err != nil {
                        return nil, err
                }
                val, err := p.expect(TOKEN_NUMBER, "strain value")
                if err != nil {
                        return nil, err
                }
                n, err := strconv.Atoi(val.Lexeme)
                if err != nil || n < 0 {
                        return nil, fmt.Errorf("%s:%d:%d: strain value must be non-negative integer", p.file, val.Line, val.Col)
                }
                switch key.Lexeme {
                case "max_branch":
                        budget.MaxBranch = n
                case "max_loop":
                        budget.MaxLoop = n
                case "max_call_depth":
                        budget.MaxCallDepth = n
                case "max_commit":
                        budget.MaxCommit = n
                case "max_nest":
                        budget.MaxNest = n
                default:
                        return nil, fmt.Errorf("%s:%d:%d: unknown strain key %q", p.file, key.Line, key.Col, key.Lexeme)
                }
                if p.check(TOKEN_COMMA) {
                        p.advance()
                }
        }
        if _, err := p.expect(TOKEN_RBRACE, "strain block"); err != nil {
                return nil, err
        }
        if _, err := p.expect(TOKEN_SEMICOLON, "strain statement"); err != nil {
                return nil, err
        }
        return &StrainStmt{
                Line: tok.Line, Col: tok.Col,
                MaxBranch:    budget.MaxBranch,
                MaxLoop:      budget.MaxLoop,
                MaxCallDepth: budget.MaxCallDepth,
                MaxCommit:    budget.MaxCommit,
                MaxNest:      budget.MaxNest,
        }, nil
}

func (p *Parser) parseCommitStmt() (Stmt, error) {
        tok := p.advance() // consume 'commit'
        label, err := p.expect(TOKEN_STRING, "commit label")
        if err != nil {
                return nil, err
        }
        body, err := p.parseBlock()
        if err != nil {
                return nil, err
        }
        return &CommitStmt{Line: tok.Line, Col: tok.Col, Label: label.Lexeme, Body: body}, nil
}

func (p *Parser) parseBlock() ([]Stmt, error) {
        if _, err := p.expect(TOKEN_LBRACE, "block"); err != nil {
                return nil, err
        }
        var stmts []Stmt
        for !p.check(TOKEN_RBRACE) && !p.check(TOKEN_EOF) {
                stmt, err := p.parseStmt()
                if err != nil {
                        return nil, err
                }
                stmts = append(stmts, stmt)
        }
        if _, err := p.expect(TOKEN_RBRACE, "block"); err != nil {
                return nil, err
        }
        return stmts, nil
}

// Expression parsing (Pratt-style)
func (p *Parser) parseExpr() (Expr, error) {
        return p.parseOr()
}

func (p *Parser) parseOr() (Expr, error) {
        left, err := p.parseAnd()
        if err != nil {
                return nil, err
        }
        for p.check(TOKEN_OR) {
                tok := p.advance()
                right, err := p.parseAnd()
                if err != nil {
                        return nil, err
                }
                left = &BinaryExpr{Line: tok.Line, Col: tok.Col, Op: "or", Left: left, Right: right}
        }
        return left, nil
}

func (p *Parser) parseAnd() (Expr, error) {
        left, err := p.parseEquality()
        if err != nil {
                return nil, err
        }
        for p.check(TOKEN_AND) {
                tok := p.advance()
                right, err := p.parseEquality()
                if err != nil {
                        return nil, err
                }
                left = &BinaryExpr{Line: tok.Line, Col: tok.Col, Op: "and", Left: left, Right: right}
        }
        return left, nil
}

func (p *Parser) parseEquality() (Expr, error) {
        left, err := p.parseComparison()
        if err != nil {
                return nil, err
        }
        for p.check(TOKEN_EQEQ) || p.check(TOKEN_NEQ) {
                tok := p.advance()
                right, err := p.parseComparison()
                if err != nil {
                        return nil, err
                }
                left = &BinaryExpr{Line: tok.Line, Col: tok.Col, Op: tok.Lexeme, Left: left, Right: right}
        }
        return left, nil
}

func (p *Parser) parseComparison() (Expr, error) {
        left, err := p.parseAddSub()
        if err != nil {
                return nil, err
        }
        for p.check(TOKEN_LT) || p.check(TOKEN_GT) || p.check(TOKEN_LE) || p.check(TOKEN_GE) {
                tok := p.advance()
                right, err := p.parseAddSub()
                if err != nil {
                        return nil, err
                }
                left = &BinaryExpr{Line: tok.Line, Col: tok.Col, Op: tok.Lexeme, Left: left, Right: right}
        }
        return left, nil
}

func (p *Parser) parseAddSub() (Expr, error) {
        left, err := p.parseMulDiv()
        if err != nil {
                return nil, err
        }
        for p.check(TOKEN_PLUS) || p.check(TOKEN_MINUS) {
                tok := p.advance()
                right, err := p.parseMulDiv()
                if err != nil {
                        return nil, err
                }
                left = &BinaryExpr{Line: tok.Line, Col: tok.Col, Op: tok.Lexeme, Left: left, Right: right}
        }
        return left, nil
}

func (p *Parser) parseMulDiv() (Expr, error) {
        left, err := p.parseUnary()
        if err != nil {
                return nil, err
        }
        for p.check(TOKEN_STAR) || p.check(TOKEN_SLASH) {
                tok := p.advance()
                right, err := p.parseUnary()
                if err != nil {
                        return nil, err
                }
                left = &BinaryExpr{Line: tok.Line, Col: tok.Col, Op: tok.Lexeme, Left: left, Right: right}
        }
        return left, nil
}

func (p *Parser) parseUnary() (Expr, error) {
        if p.check(TOKEN_MINUS) {
                tok := p.advance()
                right, err := p.parseUnary()
                if err != nil {
                        return nil, err
                }
                return &UnaryExpr{Line: tok.Line, Col: tok.Col, Op: "-", Right: right}, nil
        }
        if p.check(TOKEN_BANG) {
                tok := p.advance()
                right, err := p.parseUnary()
                if err != nil {
                        return nil, err
                }
                return &UnaryExpr{Line: tok.Line, Col: tok.Col, Op: "!", Right: right}, nil
        }
        return p.parsePostfix()
}

func (p *Parser) parsePostfix() (Expr, error) {
        expr, err := p.parsePrimary()
        if err != nil {
                return nil, err
        }
        for {
                if p.check(TOKEN_LBRACKET) {
                        tok := p.advance()
                        idx, err := p.parseExpr()
                        if err != nil {
                                return nil, err
                        }
                        if _, err := p.expect(TOKEN_RBRACKET, "index expression"); err != nil {
                                return nil, err
                        }
                        expr = &IndexExpr{Line: tok.Line, Col: tok.Col, Object: expr, Index: idx}
                } else if p.check(TOKEN_DOT) {
                        tok := p.advance()
                        field, err := p.expect(TOKEN_IDENT, "dot access")
                        if err != nil {
                                return nil, err
                        }
                        expr = &DotExpr{Line: tok.Line, Col: tok.Col, Object: expr, Field: field.Lexeme}
                } else if p.check(TOKEN_LPAREN) {
                        tok := p.advance()
                        var args []Expr
                        for !p.check(TOKEN_RPAREN) && !p.check(TOKEN_EOF) {
                                arg, err := p.parseExpr()
                                if err != nil {
                                        return nil, err
                                }
                                args = append(args, arg)
                                if !p.check(TOKEN_RPAREN) {
                                        if _, err := p.expect(TOKEN_COMMA, "call args"); err != nil {
                                                return nil, err
                                        }
                                }
                        }
                        if _, err := p.expect(TOKEN_RPAREN, "call args"); err != nil {
                                return nil, err
                        }
                        expr = &CallExpr{Line: tok.Line, Col: tok.Col, Callee: expr, Args: args}
                } else {
                        break
                }
        }
        return expr, nil
}

func (p *Parser) parsePrimary() (Expr, error) {
        tok := p.peek()
        switch tok.Type {
        case TOKEN_NUMBER:
                p.advance()
                r := new(big.Rat)
                _, ok := r.SetString(tok.Lexeme)
                if !ok {
                        return nil, fmt.Errorf("%s:%d:%d: invalid number %q", p.file, tok.Line, tok.Col, tok.Lexeme)
                }
                return &NumberLit{Line: tok.Line, Col: tok.Col, Value: r}, nil
        case TOKEN_STRING:
                p.advance()
                return &StringLit{Line: tok.Line, Col: tok.Col, Value: tok.Lexeme}, nil
        case TOKEN_TRUE:
                p.advance()
                return &BoolLit{Line: tok.Line, Col: tok.Col, Value: true}, nil
        case TOKEN_FALSE:
                p.advance()
                return &BoolLit{Line: tok.Line, Col: tok.Col, Value: false}, nil
        case TOKEN_NIL:
                p.advance()
                return &NilLit{Line: tok.Line, Col: tok.Col}, nil
        case TOKEN_IDENT:
                p.advance()
                return &Identifier{Line: tok.Line, Col: tok.Col, Name: tok.Lexeme}, nil
        case TOKEN_LPAREN:
                p.advance()
                expr, err := p.parseExpr()
                if err != nil {
                        return nil, err
                }
                if _, err := p.expect(TOKEN_RPAREN, "grouped expression"); err != nil {
                        return nil, err
                }
                return expr, nil
        case TOKEN_LBRACKET:
                return p.parseListLit()
        case TOKEN_LBRACE:
                return p.parseDictLit()
        default:
                return nil, fmt.Errorf("%s:%d:%d: unexpected token %q in expression", p.file, tok.Line, tok.Col, tok.Lexeme)
        }
}

func (p *Parser) parseListLit() (Expr, error) {
        tok := p.advance() // consume '['
        var elems []Expr
        for !p.check(TOKEN_RBRACKET) && !p.check(TOKEN_EOF) {
                elem, err := p.parseExpr()
                if err != nil {
                        return nil, err
                }
                elems = append(elems, elem)
                if !p.check(TOKEN_RBRACKET) {
                        if _, err := p.expect(TOKEN_COMMA, "list literal"); err != nil {
                                return nil, err
                        }
                }
        }
        if _, err := p.expect(TOKEN_RBRACKET, "list literal"); err != nil {
                return nil, err
        }
        return &ListLit{Line: tok.Line, Col: tok.Col, Elements: elems}, nil
}

func (p *Parser) parseDictLit() (Expr, error) {
        tok := p.advance() // consume '{'
        var keys []string
        var vals []Expr
        for !p.check(TOKEN_RBRACE) && !p.check(TOKEN_EOF) {
                key, err := p.expect(TOKEN_STRING, "dict key")
                if err != nil {
                        return nil, err
                }
                if _, err := p.expect(TOKEN_COLON, "dict value"); err != nil {
                        return nil, err
                }
                val, err := p.parseExpr()
                if err != nil {
                        return nil, err
                }
                keys = append(keys, key.Lexeme)
                vals = append(vals, val)
                if !p.check(TOKEN_RBRACE) {
                        if _, err := p.expect(TOKEN_COMMA, "dict literal"); err != nil {
                                return nil, err
                        }
                }
        }
        if _, err := p.expect(TOKEN_RBRACE, "dict literal"); err != nil {
                return nil, err
        }
        return &DictLit{Line: tok.Line, Col: tok.Col, Keys: keys, Values: vals}, nil
}

func tokenTypeName(t TokenType) string {
        switch t {
        case TOKEN_NUMBER:
                return "number"
        case TOKEN_STRING:
                return "string"
        case TOKEN_IDENT:
                return "identifier"
        case TOKEN_SEMICOLON:
                return "';'"
        case TOKEN_LPAREN:
                return "'('"
        case TOKEN_RPAREN:
                return "')'"
        case TOKEN_LBRACE:
                return "'{'"
        case TOKEN_RBRACE:
                return "'}'"
        case TOKEN_LBRACKET:
                return "'['"
        case TOKEN_RBRACKET:
                return "']'"
        case TOKEN_COMMA:
                return "','"
        case TOKEN_COLON:
                return "':'"
        case TOKEN_ASSIGN:
                return "'='"
        case TOKEN_AS:
                return "'as'"
        default:
                return fmt.Sprintf("token(%d)", t)
        }
}
