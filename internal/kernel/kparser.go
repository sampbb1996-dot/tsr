package kernel

import (
        "fmt"
        "strconv"
)

type KParser struct {
        tokens []KToken
        pos    int
        file   string
}

func NewKParser(tokens []KToken, file string) *KParser {
        return &KParser{tokens: tokens, file: file}
}

func (p *KParser) Parse() (*KProgram, error) {
        prog := &KProgram{}
        p.skipNewlines()
        for !p.check(KT_EOF) {
                stmt, err := p.parseStmt()
                if err != nil {
                        return nil, err
                }
                if stmt != nil {
                        prog.Stmts = append(prog.Stmts, stmt)
                }
                p.skipNewlines()
        }
        return prog, nil
}

func (p *KParser) peek() KToken {
        return p.tokens[p.pos]
}

func (p *KParser) check(t KTokenType) bool {
        return p.tokens[p.pos].Type == t
}

func (p *KParser) advance() KToken {
        tok := p.tokens[p.pos]
        if tok.Type != KT_EOF {
                p.pos++
        }
        return tok
}

func (p *KParser) expect(t KTokenType, ctx string) (KToken, error) {
        if !p.check(t) {
                tok := p.peek()
                return KToken{}, fmt.Errorf("%s:%d:%d: kernel: expected %d in %s, got %q", p.file, tok.Line, tok.Col, t, ctx, tok.Lexeme)
        }
        return p.advance(), nil
}

func (p *KParser) skipNewlines() {
        for p.check(KT_NEWLINE) {
                p.advance()
        }
}

func (p *KParser) expectNewlineOrEOF() error {
        if p.check(KT_EOF) {
                return nil
        }
        if p.check(KT_NEWLINE) {
                p.advance()
                return nil
        }
        tok := p.peek()
        return fmt.Errorf("%s:%d:%d: kernel: expected newline, got %q", p.file, tok.Line, tok.Col, tok.Lexeme)
}

func (p *KParser) parseStmt() (KStmt, error) {
        tok := p.peek()
        if tok.Type != KT_IDENT {
                if tok.Type == KT_NEWLINE || tok.Type == KT_EOF {
                        p.advance()
                        return nil, nil
                }
                return nil, fmt.Errorf("%s:%d:%d: kernel: unexpected token %q", p.file, tok.Line, tok.Col, tok.Lexeme)
        }

        switch tok.Lexeme {
        case "mode":
                return p.parseModeStmt()
        case "budget":
                return p.parseBudgetStmt()
        case "set":
                return p.parseSetStmt()
        case "say":
                return p.parseSayStmt()
        case "if":
                return p.parseIfStmt()
        case "while":
                return p.parseWhileStmt()
        case "fn":
                return p.parseFnStmt()
        case "return":
                return p.parseReturnStmt()
        case "read_file":
                return p.parseReadFileStmt()
        case "write_file":
                return p.parseWriteFileStmt()
        case "http_get":
                return p.parseHttpGetStmt()
        case "import":
                return p.parseImportStmt()
        case "context":
                return p.parseContextStmt()
        default:
                return nil, fmt.Errorf("%s:%d:%d: kernel: unknown statement %q", p.file, tok.Line, tok.Col, tok.Lexeme)
        }
}

func (p *KParser) parseModeStmt() (KStmt, error) {
        tok := p.advance() // consume 'mode'
        modeTok, err := p.expect(KT_STRING, "mode")
        if err != nil {
                return nil, err
        }
        if modeTok.Lexeme != "CALM" && modeTok.Lexeme != "READONLY" {
                return nil, fmt.Errorf("%s:%d:%d: kernel: mode must be CALM or READONLY", p.file, tok.Line, tok.Col)
        }
        if err := p.expectNewlineOrEOF(); err != nil {
                return nil, err
        }
        return &KModeStmt{Line: tok.Line, Col: tok.Col, Mode: modeTok.Lexeme}, nil
}

func (p *KParser) parseBudgetStmt() (KStmt, error) {
        tok := p.advance() // consume 'budget'
        if p.check(KT_IDENT) && p.peek().Lexeme == "auto" {
                p.advance()
                if err := p.expectNewlineOrEOF(); err != nil {
                        return nil, err
                }
                return &KBudgetAutoStmt{Line: tok.Line, Col: tok.Col}, nil
        }
        // budget { max_branch: N, ... }
        if _, err := p.expect(KT_LBRACE, "budget"); err != nil {
                return nil, err
        }
        budget := KBudgetStmt{
                Line:         tok.Line,
                Col:          tok.Col,
                MaxBranch:    20,
                MaxLoop:      20,
                MaxCallDepth: 50,
                MaxCommit:    50,
                MaxNest:      20,
        }
        p.skipNewlines()
        for !p.check(KT_RBRACE) && !p.check(KT_EOF) {
                key, err := p.expect(KT_IDENT, "budget key")
                if err != nil {
                        return nil, err
                }
                if _, err := p.expect(KT_COLON, "budget value"); err != nil {
                        return nil, err
                }
                val, err := p.expect(KT_NUMBER, "budget value")
                if err != nil {
                        return nil, err
                }
                n, err := strconv.Atoi(val.Lexeme)
                if err != nil || n < 0 {
                        return nil, fmt.Errorf("%s:%d:%d: kernel: budget value must be non-negative int", p.file, val.Line, val.Col)
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
                        return nil, fmt.Errorf("%s:%d:%d: kernel: unknown budget key %q", p.file, key.Line, key.Col, key.Lexeme)
                }
                if p.check(KT_COMMA) {
                        p.advance()
                }
                p.skipNewlines()
        }
        if _, err := p.expect(KT_RBRACE, "budget"); err != nil {
                return nil, err
        }
        if err := p.expectNewlineOrEOF(); err != nil {
                return nil, err
        }
        return &budget, nil
}

func (p *KParser) parseSetStmt() (KStmt, error) {
        tok := p.advance() // consume 'set'
        name, err := p.expect(KT_IDENT, "set name")
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(KT_ASSIGN, "set ="); err != nil {
                return nil, err
        }
        val, err := p.parseExpr()
        if err != nil {
                return nil, err
        }
        if err := p.expectNewlineOrEOF(); err != nil {
                return nil, err
        }
        return &KSetStmt{Line: tok.Line, Col: tok.Col, Name: name.Lexeme, Value: val}, nil
}

func (p *KParser) parseSayStmt() (KStmt, error) {
        tok := p.advance() // consume 'say'
        val, err := p.parseExpr()
        if err != nil {
                return nil, err
        }
        if err := p.expectNewlineOrEOF(); err != nil {
                return nil, err
        }
        return &KSayStmt{Line: tok.Line, Col: tok.Col, Value: val}, nil
}

func (p *KParser) parseIfStmt() (KStmt, error) {
        tok := p.advance() // consume 'if'
        cond, err := p.parseExpr()
        if err != nil {
                return nil, err
        }
        p.skipNewlines()
        if _, err := p.expect(KT_LBRACE, "if body"); err != nil {
                return nil, err
        }
        p.skipNewlines()
        consequence, err := p.parseBlock()
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(KT_RBRACE, "if body"); err != nil {
                return nil, err
        }

        var alternative []KStmt
        p.skipNewlines()
        if p.check(KT_IDENT) && p.peek().Lexeme == "else" {
                p.advance()
                p.skipNewlines()
                if _, err := p.expect(KT_LBRACE, "else body"); err != nil {
                        return nil, err
                }
                p.skipNewlines()
                alternative, err = p.parseBlock()
                if err != nil {
                        return nil, err
                }
                if _, err := p.expect(KT_RBRACE, "else body"); err != nil {
                        return nil, err
                }
                p.skipNewlines()
        }
        return &KIfStmt{Line: tok.Line, Col: tok.Col, Condition: cond, Consequence: consequence, Alternative: alternative}, nil
}

func (p *KParser) parseWhileStmt() (KStmt, error) {
        tok := p.advance() // consume 'while'
        cond, err := p.parseExpr()
        if err != nil {
                return nil, err
        }
        p.skipNewlines()
        if _, err := p.expect(KT_LBRACE, "while body"); err != nil {
                return nil, err
        }
        p.skipNewlines()
        body, err := p.parseBlock()
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(KT_RBRACE, "while body"); err != nil {
                return nil, err
        }
        p.skipNewlines()
        return &KWhileStmt{Line: tok.Line, Col: tok.Col, Condition: cond, Body: body}, nil
}

func (p *KParser) parseFnStmt() (KStmt, error) {
        tok := p.advance() // consume 'fn'
        name, err := p.expect(KT_IDENT, "fn name")
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(KT_LPAREN, "fn params"); err != nil {
                return nil, err
        }
        var params []string
        for !p.check(KT_RPAREN) && !p.check(KT_EOF) {
                param, err := p.expect(KT_IDENT, "fn param")
                if err != nil {
                        return nil, err
                }
                params = append(params, param.Lexeme)
                if !p.check(KT_RPAREN) {
                        if _, err := p.expect(KT_COMMA, "fn params"); err != nil {
                                return nil, err
                        }
                }
        }
        if _, err := p.expect(KT_RPAREN, "fn params"); err != nil {
                return nil, err
        }
        p.skipNewlines()
        if _, err := p.expect(KT_LBRACE, "fn body"); err != nil {
                return nil, err
        }
        p.skipNewlines()
        body, err := p.parseBlock()
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(KT_RBRACE, "fn body"); err != nil {
                return nil, err
        }
        p.skipNewlines()
        return &KFnStmt{Line: tok.Line, Col: tok.Col, Name: name.Lexeme, Params: params, Body: body}, nil
}

func (p *KParser) parseReturnStmt() (KStmt, error) {
        tok := p.advance() // consume 'return'
        var val KExpr
        if !p.check(KT_NEWLINE) && !p.check(KT_EOF) {
                var err error
                val, err = p.parseExpr()
                if err != nil {
                        return nil, err
                }
        }
        if val == nil {
                val = &KNilLit{Line: tok.Line, Col: tok.Col}
        }
        if err := p.expectNewlineOrEOF(); err != nil {
                return nil, err
        }
        return &KReturnStmt{Line: tok.Line, Col: tok.Col, Value: val}, nil
}

func (p *KParser) parseReadFileStmt() (KStmt, error) {
        tok := p.advance() // consume 'read_file'
        pathTok, err := p.expect(KT_STRING, "read_file path")
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(KT_ARROW, "read_file ->"); err != nil {
                return nil, err
        }
        varTok, err := p.expect(KT_IDENT, "read_file varname")
        if err != nil {
                return nil, err
        }
        if err := p.expectNewlineOrEOF(); err != nil {
                return nil, err
        }
        return &KReadFileStmt{Line: tok.Line, Col: tok.Col, Path: pathTok.Lexeme, VarName: varTok.Lexeme}, nil
}

func (p *KParser) parseWriteFileStmt() (KStmt, error) {
        tok := p.advance() // consume 'write_file'
        pathTok, err := p.expect(KT_STRING, "write_file path")
        if err != nil {
                return nil, err
        }
        val, err := p.parseExpr()
        if err != nil {
                return nil, err
        }
        if err := p.expectNewlineOrEOF(); err != nil {
                return nil, err
        }
        return &KWriteFileStmt{Line: tok.Line, Col: tok.Col, Path: pathTok.Lexeme, Value: val}, nil
}

func (p *KParser) parseHttpGetStmt() (KStmt, error) {
        tok := p.advance() // consume 'http_get'
        urlTok, err := p.expect(KT_STRING, "http_get url")
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(KT_ARROW, "http_get ->"); err != nil {
                return nil, err
        }
        varTok, err := p.expect(KT_IDENT, "http_get varname")
        if err != nil {
                return nil, err
        }
        if err := p.expectNewlineOrEOF(); err != nil {
                return nil, err
        }
        return &KHttpGetStmt{Line: tok.Line, Col: tok.Col, URL: urlTok.Lexeme, VarName: varTok.Lexeme}, nil
}

func (p *KParser) parseImportStmt() (KStmt, error) {
        tok := p.advance() // consume 'import'
        pathTok, err := p.expect(KT_STRING, "import path")
        if err != nil {
                return nil, err
        }
        if _, err := p.expect(KT_IDENT, "import 'as'"); err != nil {
                return nil, err
        }
        aliasTok, err := p.expect(KT_IDENT, "import alias")
        if err != nil {
                return nil, err
        }
        if err := p.expectNewlineOrEOF(); err != nil {
                return nil, err
        }
        return &KImportStmt{Line: tok.Line, Col: tok.Col, Path: pathTok.Lexeme, Alias: aliasTok.Lexeme}, nil
}

func (p *KParser) parseBlock() ([]KStmt, error) {
        var stmts []KStmt
        for !p.check(KT_RBRACE) && !p.check(KT_EOF) {
                p.skipNewlines()
                if p.check(KT_RBRACE) {
                        break
                }
                stmt, err := p.parseStmt()
                if err != nil {
                        return nil, err
                }
                if stmt != nil {
                        stmts = append(stmts, stmt)
                }
        }
        return stmts, nil
}

// Expression parsing
func (p *KParser) parseExpr() (KExpr, error) {
        return p.parseOr()
}

func (p *KParser) parseOr() (KExpr, error) {
        left, err := p.parseAnd()
        if err != nil {
                return nil, err
        }
        for p.check(KT_OR) {
                tok := p.advance()
                right, err := p.parseAnd()
                if err != nil {
                        return nil, err
                }
                left = &KBinaryExpr{Line: tok.Line, Col: tok.Col, Op: "or", Left: left, Right: right}
        }
        return left, nil
}

func (p *KParser) parseAnd() (KExpr, error) {
        left, err := p.parseEquality()
        if err != nil {
                return nil, err
        }
        for p.check(KT_AND) {
                tok := p.advance()
                right, err := p.parseEquality()
                if err != nil {
                        return nil, err
                }
                left = &KBinaryExpr{Line: tok.Line, Col: tok.Col, Op: "and", Left: left, Right: right}
        }
        return left, nil
}

func (p *KParser) parseEquality() (KExpr, error) {
        left, err := p.parseComparison()
        if err != nil {
                return nil, err
        }
        for p.check(KT_EQEQ) || p.check(KT_NEQ) {
                tok := p.advance()
                right, err := p.parseComparison()
                if err != nil {
                        return nil, err
                }
                left = &KBinaryExpr{Line: tok.Line, Col: tok.Col, Op: tok.Lexeme, Left: left, Right: right}
        }
        return left, nil
}

func (p *KParser) parseComparison() (KExpr, error) {
        left, err := p.parseAddSub()
        if err != nil {
                return nil, err
        }
        for p.check(KT_LT) || p.check(KT_GT) || p.check(KT_LE) || p.check(KT_GE) {
                tok := p.advance()
                right, err := p.parseAddSub()
                if err != nil {
                        return nil, err
                }
                left = &KBinaryExpr{Line: tok.Line, Col: tok.Col, Op: tok.Lexeme, Left: left, Right: right}
        }
        return left, nil
}

func (p *KParser) parseAddSub() (KExpr, error) {
        left, err := p.parseMulDiv()
        if err != nil {
                return nil, err
        }
        for p.check(KT_PLUS) || p.check(KT_MINUS) {
                tok := p.advance()
                right, err := p.parseMulDiv()
                if err != nil {
                        return nil, err
                }
                left = &KBinaryExpr{Line: tok.Line, Col: tok.Col, Op: tok.Lexeme, Left: left, Right: right}
        }
        return left, nil
}

func (p *KParser) parseMulDiv() (KExpr, error) {
        left, err := p.parseUnary()
        if err != nil {
                return nil, err
        }
        for p.check(KT_STAR) || p.check(KT_SLASH) {
                tok := p.advance()
                right, err := p.parseMulDiv()
                if err != nil {
                        return nil, err
                }
                left = &KBinaryExpr{Line: tok.Line, Col: tok.Col, Op: tok.Lexeme, Left: left, Right: right}
        }
        return left, nil
}

func (p *KParser) parseUnary() (KExpr, error) {
        if p.check(KT_MINUS) {
                tok := p.advance()
                right, err := p.parseUnary()
                if err != nil {
                        return nil, err
                }
                return &KUnaryExpr{Line: tok.Line, Col: tok.Col, Op: "-", Right: right}, nil
        }
        if p.check(KT_BANG) {
                tok := p.advance()
                right, err := p.parseUnary()
                if err != nil {
                        return nil, err
                }
                return &KUnaryExpr{Line: tok.Line, Col: tok.Col, Op: "!", Right: right}, nil
        }
        return p.parsePrimary()
}

func (p *KParser) parsePrimary() (KExpr, error) {
        tok := p.peek()
        switch tok.Type {
        case KT_NUMBER:
                p.advance()
                return &KNumberLit{Line: tok.Line, Col: tok.Col, Raw: tok.Lexeme}, nil
        case KT_STRING:
                p.advance()
                return &KStringLit{Line: tok.Line, Col: tok.Col, Value: tok.Lexeme}, nil
        case KT_TRUE:
                p.advance()
                return &KBoolLit{Line: tok.Line, Col: tok.Col, Value: true}, nil
        case KT_FALSE:
                p.advance()
                return &KBoolLit{Line: tok.Line, Col: tok.Col, Value: false}, nil
        case KT_NIL:
                p.advance()
                return &KNilLit{Line: tok.Line, Col: tok.Col}, nil
        case KT_IDENT:
                p.advance()
                // check for call
                if p.check(KT_LPAREN) {
                        p.advance()
                        var args []KExpr
                        for !p.check(KT_RPAREN) && !p.check(KT_EOF) {
                                arg, err := p.parseExpr()
                                if err != nil {
                                        return nil, err
                                }
                                args = append(args, arg)
                                if !p.check(KT_RPAREN) {
                                        if _, err := p.expect(KT_COMMA, "call args"); err != nil {
                                                return nil, err
                                        }
                                }
                        }
                        if _, err := p.expect(KT_RPAREN, "call args"); err != nil {
                                return nil, err
                        }
                        return &KCallExpr{Line: tok.Line, Col: tok.Col, Callee: tok.Lexeme, Args: args}, nil
                }
                return &KIdent{Line: tok.Line, Col: tok.Col, Name: tok.Lexeme}, nil
        case KT_LBRACKET:
                return p.parseListLit()
        case KT_LBRACE:
                return p.parseDictLit()
        case KT_LPAREN:
                p.advance()
                expr, err := p.parseExpr()
                if err != nil {
                        return nil, err
                }
                if _, err := p.expect(KT_RPAREN, "grouped expr"); err != nil {
                        return nil, err
                }
                return expr, nil
        default:
                return nil, fmt.Errorf("%s:%d:%d: kernel: unexpected token %q in expression", p.file, tok.Line, tok.Col, tok.Lexeme)
        }
}

func (p *KParser) parseListLit() (KExpr, error) {
        tok := p.advance()
        var elems []KExpr
        for !p.check(KT_RBRACKET) && !p.check(KT_EOF) {
                elem, err := p.parseExpr()
                if err != nil {
                        return nil, err
                }
                elems = append(elems, elem)
                if !p.check(KT_RBRACKET) {
                        if _, err := p.expect(KT_COMMA, "list"); err != nil {
                                return nil, err
                        }
                }
        }
        if _, err := p.expect(KT_RBRACKET, "list"); err != nil {
                return nil, err
        }
        return &KListLit{Line: tok.Line, Col: tok.Col, Elements: elems}, nil
}

func (p *KParser) parseDictLit() (KExpr, error) {
        tok := p.advance()
        var keys []string
        var vals []KExpr
        for !p.check(KT_RBRACE) && !p.check(KT_EOF) {
                key, err := p.expect(KT_STRING, "dict key")
                if err != nil {
                        return nil, err
                }
                if _, err := p.expect(KT_COLON, "dict value"); err != nil {
                        return nil, err
                }
                val, err := p.parseExpr()
                if err != nil {
                        return nil, err
                }
                keys = append(keys, key.Lexeme)
                vals = append(vals, val)
                if !p.check(KT_RBRACE) {
                        if _, err := p.expect(KT_COMMA, "dict"); err != nil {
                                return nil, err
                        }
                }
        }
        if _, err := p.expect(KT_RBRACE, "dict"); err != nil {
                return nil, err
        }
        return &KDictLit{Line: tok.Line, Col: tok.Col, Keys: keys, Values: vals}, nil
}

// parseContextStmt parses:
//
//      context {
//        env: "dev"
//        risk: "low"
//        actor: "service:x"
//      }
func (p *KParser) parseContextStmt() (KStmt, error) {
        tok := p.advance() // consume 'context'
        if _, err := p.expect(KT_LBRACE, "context block"); err != nil {
                return nil, err
        }
        ctx := &KContextStmt{Line: tok.Line, Col: tok.Col}
        p.skipNewlines()
        for !p.check(KT_RBRACE) && !p.check(KT_EOF) {
                key, err := p.expect(KT_IDENT, "context key")
                if err != nil {
                        return nil, err
                }
                if _, err := p.expect(KT_COLON, "context value"); err != nil {
                        return nil, err
                }
                val, err := p.expect(KT_STRING, "context value")
                if err != nil {
                        return nil, err
                }
                switch key.Lexeme {
                case "env":
                        if val.Lexeme != "dev" && val.Lexeme != "staging" && val.Lexeme != "prod" {
                                return nil, fmt.Errorf("%s:%d:%d: kernel: context env must be dev, staging, or prod",
                                        p.file, val.Line, val.Col)
                        }
                        ctx.Env = val.Lexeme
                case "risk":
                        if val.Lexeme != "low" && val.Lexeme != "medium" && val.Lexeme != "high" {
                                return nil, fmt.Errorf("%s:%d:%d: kernel: context risk must be low, medium, or high",
                                        p.file, val.Line, val.Col)
                        }
                        ctx.Risk = val.Lexeme
                case "actor":
                        ctx.Actor = val.Lexeme
                default:
                        return nil, fmt.Errorf("%s:%d:%d: kernel: unknown context key %q",
                                p.file, key.Line, key.Col, key.Lexeme)
                }
                p.skipNewlines()
        }
        if _, err := p.expect(KT_RBRACE, "context block"); err != nil {
                return nil, err
        }
        p.skipNewlines()
        return ctx, nil
}
