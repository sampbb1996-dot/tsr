package kernel

// KNode is the base interface for all KERNEL AST nodes
type KNode interface {
        kNodePos() (line, col int)
}

type KProgram struct {
        Stmts []KStmt
}

type KStmt interface {
        KNode
        kStmtNode()
}

type (
        KModeStmt struct {
                Line, Col int
                Mode      string
        }

        KBudgetAutoStmt struct {
                Line, Col int
        }

        KBudgetStmt struct {
                Line, Col    int
                MaxBranch    int
                MaxLoop      int
                MaxCallDepth int
                MaxCommit    int
                MaxNest      int
        }

        KSetStmt struct {
                Line, Col int
                Name      string
                Value     KExpr
        }

        KSayStmt struct {
                Line, Col int
                Value     KExpr
        }

        KIfStmt struct {
                Line, Col   int
                Condition   KExpr
                Consequence []KStmt
                Alternative []KStmt
        }

        KWhileStmt struct {
                Line, Col int
                Condition KExpr
                Body      []KStmt
        }

        KFnStmt struct {
                Line, Col int
                Name      string
                Params    []string
                Body      []KStmt
        }

        KReturnStmt struct {
                Line, Col int
                Value     KExpr
        }

        KReadFileStmt struct {
                Line, Col int
                Path      string
                VarName   string
        }

        KWriteFileStmt struct {
                Line, Col int
                Path      string
                Value     KExpr
        }

        KHttpGetStmt struct {
                Line, Col int
                URL       string
                VarName   string
        }

        KImportStmt struct {
                Line, Col int
                Path      string
                Alias     string
        }

        // KContextStmt declares explicit environmental state for policy evaluation.
        // When present, the compiler calls policy.Evaluate to derive regime/strain/capabilities.
        KContextStmt struct {
                Line, Col int
                Env       string
                Risk      string
                Actor     string
        }
)

func (s *KModeStmt) kStmtNode()      {}
func (s *KBudgetAutoStmt) kStmtNode() {}
func (s *KBudgetStmt) kStmtNode()    {}
func (s *KSetStmt) kStmtNode()       {}
func (s *KSayStmt) kStmtNode()       {}
func (s *KIfStmt) kStmtNode()        {}
func (s *KWhileStmt) kStmtNode()     {}
func (s *KFnStmt) kStmtNode()        {}
func (s *KReturnStmt) kStmtNode()    {}
func (s *KReadFileStmt) kStmtNode()  {}
func (s *KWriteFileStmt) kStmtNode() {}
func (s *KHttpGetStmt) kStmtNode()   {}
func (s *KImportStmt) kStmtNode()    {}
func (s *KContextStmt) kStmtNode()   {}

func (s *KModeStmt) kNodePos() (int, int)       { return s.Line, s.Col }
func (s *KBudgetAutoStmt) kNodePos() (int, int) { return s.Line, s.Col }
func (s *KBudgetStmt) kNodePos() (int, int)    { return s.Line, s.Col }
func (s *KSetStmt) kNodePos() (int, int)       { return s.Line, s.Col }
func (s *KSayStmt) kNodePos() (int, int)       { return s.Line, s.Col }
func (s *KIfStmt) kNodePos() (int, int)        { return s.Line, s.Col }
func (s *KWhileStmt) kNodePos() (int, int)     { return s.Line, s.Col }
func (s *KFnStmt) kNodePos() (int, int)        { return s.Line, s.Col }
func (s *KReturnStmt) kNodePos() (int, int)    { return s.Line, s.Col }
func (s *KReadFileStmt) kNodePos() (int, int)  { return s.Line, s.Col }
func (s *KWriteFileStmt) kNodePos() (int, int) { return s.Line, s.Col }
func (s *KHttpGetStmt) kNodePos() (int, int)   { return s.Line, s.Col }
func (s *KImportStmt) kNodePos() (int, int)    { return s.Line, s.Col }
func (s *KContextStmt) kNodePos() (int, int)   { return s.Line, s.Col }

// Expressions (subset of TSR)
type KExpr interface {
        KNode
        kExprNode()
}

type (
        KNumberLit struct {
                Line, Col int
                Raw       string
        }
        KStringLit struct {
                Line, Col int
                Value     string
        }
        KBoolLit struct {
                Line, Col int
                Value     bool
        }
        KNilLit struct {
                Line, Col int
        }
        KIdent struct {
                Line, Col int
                Name      string
        }
        KListLit struct {
                Line, Col int
                Elements  []KExpr
        }
        KDictLit struct {
                Line, Col int
                Keys      []string
                Values    []KExpr
        }
        KBinaryExpr struct {
                Line, Col int
                Op        string
                Left      KExpr
                Right     KExpr
        }
        KUnaryExpr struct {
                Line, Col int
                Op        string
                Right     KExpr
        }
        KCallExpr struct {
                Line, Col int
                Callee    string
                Args      []KExpr
        }
)

func (e *KNumberLit) kExprNode()  {}
func (e *KStringLit) kExprNode()  {}
func (e *KBoolLit) kExprNode()    {}
func (e *KNilLit) kExprNode()     {}
func (e *KIdent) kExprNode()      {}
func (e *KListLit) kExprNode()    {}
func (e *KDictLit) kExprNode()    {}
func (e *KBinaryExpr) kExprNode() {}
func (e *KUnaryExpr) kExprNode()  {}
func (e *KCallExpr) kExprNode()   {}

func (e *KNumberLit) kNodePos() (int, int)  { return e.Line, e.Col }
func (e *KStringLit) kNodePos() (int, int)  { return e.Line, e.Col }
func (e *KBoolLit) kNodePos() (int, int)    { return e.Line, e.Col }
func (e *KNilLit) kNodePos() (int, int)     { return e.Line, e.Col }
func (e *KIdent) kNodePos() (int, int)      { return e.Line, e.Col }
func (e *KListLit) kNodePos() (int, int)    { return e.Line, e.Col }
func (e *KDictLit) kNodePos() (int, int)    { return e.Line, e.Col }
func (e *KBinaryExpr) kNodePos() (int, int) { return e.Line, e.Col }
func (e *KUnaryExpr) kNodePos() (int, int)  { return e.Line, e.Col }
func (e *KCallExpr) kNodePos() (int, int)   { return e.Line, e.Col }
