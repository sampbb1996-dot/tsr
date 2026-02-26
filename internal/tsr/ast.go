package tsr

import "math/big"

// Node is the base interface for all AST nodes
type Node interface {
        nodePos() (line, col int)
}

// Statements
type (
        Program struct {
                Stmts []Stmt
        }

        Stmt interface {
                Node
                stmtNode()
        }

        LetStmt struct {
                Line, Col int
                Name      string
                Value     Expr
        }

        AssignStmt struct {
                Line, Col int
                Name      string
                Value     Expr
        }

        SayStmt struct {
                Line, Col int
                Value     Expr
        }

        IfStmt struct {
                Line, Col   int
                Condition   Expr
                Consequence []Stmt
                Alternative []Stmt
        }

        WhileStmt struct {
                Line, Col int
                Condition Expr
                Body      []Stmt
        }

        FnStmt struct {
                Line, Col int
                Name      string
                Params    []string
                Body      []Stmt
        }

        ReturnStmt struct {
                Line, Col int
                Value     Expr
        }

        ImportStmt struct {
                Line, Col int
                Path      string
                Alias     string
        }

        RegimeStmt struct {
                Line, Col int
                Mode      string
        }

        StrainStmt struct {
                Line, Col    int
                MaxBranch    int
                MaxLoop      int
                MaxCallDepth int
                MaxCommit    int
                MaxNest      int
        }

        CommitStmt struct {
                Line, Col int
                Label     string
                Body      []Stmt
        }

        CapabilityStmt struct {
                Line, Col int
                Kind      string
                Pattern   string
        }

        ExprStmt struct {
                Line, Col int
                Expr      Expr
        }
)

func (s *LetStmt) stmtNode()    {}
func (s *AssignStmt) stmtNode() {}
func (s *SayStmt) stmtNode()    {}
func (s *IfStmt) stmtNode()     {}
func (s *WhileStmt) stmtNode()  {}
func (s *FnStmt) stmtNode()     {}
func (s *ReturnStmt) stmtNode() {}
func (s *ImportStmt) stmtNode() {}
func (s *RegimeStmt) stmtNode() {}
func (s *StrainStmt) stmtNode() {}
func (s *CommitStmt) stmtNode()     {}
func (s *CapabilityStmt) stmtNode() {}
func (s *ExprStmt) stmtNode()       {}

func (s *LetStmt) nodePos() (int, int)    { return s.Line, s.Col }
func (s *AssignStmt) nodePos() (int, int) { return s.Line, s.Col }
func (s *SayStmt) nodePos() (int, int)    { return s.Line, s.Col }
func (s *IfStmt) nodePos() (int, int)     { return s.Line, s.Col }
func (s *WhileStmt) nodePos() (int, int)  { return s.Line, s.Col }
func (s *FnStmt) nodePos() (int, int)     { return s.Line, s.Col }
func (s *ReturnStmt) nodePos() (int, int) { return s.Line, s.Col }
func (s *ImportStmt) nodePos() (int, int) { return s.Line, s.Col }
func (s *RegimeStmt) nodePos() (int, int) { return s.Line, s.Col }
func (s *StrainStmt) nodePos() (int, int) { return s.Line, s.Col }
func (s *CommitStmt) nodePos() (int, int)     { return s.Line, s.Col }
func (s *CapabilityStmt) nodePos() (int, int) { return s.Line, s.Col }
func (s *ExprStmt) nodePos() (int, int)       { return s.Line, s.Col }

// Expressions
type (
        Expr interface {
                Node
                exprNode()
        }

        NumberLit struct {
                Line, Col int
                Value     *big.Rat
        }

        StringLit struct {
                Line, Col int
                Value     string
        }

        BoolLit struct {
                Line, Col int
                Value     bool
        }

        NilLit struct {
                Line, Col int
        }

        Identifier struct {
                Line, Col int
                Name      string
        }

        UnaryExpr struct {
                Line, Col int
                Op        string
                Right     Expr
        }

        BinaryExpr struct {
                Line, Col int
                Op        string
                Left      Expr
                Right     Expr
        }

        ListLit struct {
                Line, Col int
                Elements  []Expr
        }

        DictLit struct {
                Line, Col int
                Keys      []string
                Values    []Expr
        }

        IndexExpr struct {
                Line, Col int
                Object    Expr
                Index     Expr
        }

        CallExpr struct {
                Line, Col int
                Callee    Expr
                Args      []Expr
        }

        DotExpr struct {
                Line, Col int
                Object    Expr
                Field     string
        }
)

func (e *NumberLit) exprNode()  {}
func (e *StringLit) exprNode()  {}
func (e *BoolLit) exprNode()    {}
func (e *NilLit) exprNode()     {}
func (e *Identifier) exprNode() {}
func (e *UnaryExpr) exprNode()  {}
func (e *BinaryExpr) exprNode() {}
func (e *ListLit) exprNode()    {}
func (e *DictLit) exprNode()    {}
func (e *IndexExpr) exprNode()  {}
func (e *CallExpr) exprNode()   {}
func (e *DotExpr) exprNode()    {}

func (e *NumberLit) nodePos() (int, int)  { return e.Line, e.Col }
func (e *StringLit) nodePos() (int, int)  { return e.Line, e.Col }
func (e *BoolLit) nodePos() (int, int)    { return e.Line, e.Col }
func (e *NilLit) nodePos() (int, int)     { return e.Line, e.Col }
func (e *Identifier) nodePos() (int, int) { return e.Line, e.Col }
func (e *UnaryExpr) nodePos() (int, int)  { return e.Line, e.Col }
func (e *BinaryExpr) nodePos() (int, int) { return e.Line, e.Col }
func (e *ListLit) nodePos() (int, int)    { return e.Line, e.Col }
func (e *DictLit) nodePos() (int, int)    { return e.Line, e.Col }
func (e *IndexExpr) nodePos() (int, int)  { return e.Line, e.Col }
func (e *CallExpr) nodePos() (int, int)   { return e.Line, e.Col }
func (e *DotExpr) nodePos() (int, int)    { return e.Line, e.Col }
