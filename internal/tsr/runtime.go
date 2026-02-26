package tsr

import (
        "fmt"
        "io"
        "math/big"
        "os"
        "path/filepath"
        "strings"
)

// Value types
type ValueType int

const (
        ValNil ValueType = iota
        ValBool
        ValNumber
        ValString
        ValList
        ValDict
        ValFunc
        ValModule
)

type Value struct {
        Type    ValueType
        BoolVal bool
        NumVal  *big.Rat
        StrVal  string
        ListVal []*Value
        DictVal map[string]*Value
        FuncVal *Function
        ModVal  *Module
}

type Function struct {
        Name    string
        Params  []string
        Body    []Stmt
        Closure *Env
        Builtin func(args []*Value, gov *GovernanceState, rt *Runtime) (*Value, error)
}

var nilValue = &Value{Type: ValNil}
var trueValue = &Value{Type: ValBool, BoolVal: true}
var falseValue = &Value{Type: ValBool, BoolVal: false}

func numVal(r *big.Rat) *Value { return &Value{Type: ValNumber, NumVal: r} }
func strVal(s string) *Value   { return &Value{Type: ValString, StrVal: s} }
func boolVal(b bool) *Value {
        if b {
                return trueValue
        }
        return falseValue
}

// Env is a lexical scope
type Env struct {
        vars   map[string]*Value
        parent *Env
}

func newEnv(parent *Env) *Env {
        return &Env{vars: make(map[string]*Value), parent: parent}
}

func (e *Env) get(name string) (*Value, bool) {
        if v, ok := e.vars[name]; ok {
                return v, true
        }
        if e.parent != nil {
                return e.parent.get(name)
        }
        return nil, false
}

func (e *Env) set(name string, v *Value) {
        e.vars[name] = v
}

func (e *Env) assign(name string, v *Value) bool {
        if _, ok := e.vars[name]; ok {
                e.vars[name] = v
                return true
        }
        if e.parent != nil {
                return e.parent.assign(name, v)
        }
        return false
}

// Runtime is the main interpreter
type Runtime struct {
        file              string
        gov               *GovernanceState
        global            *Env
        stdout            io.Writer
        stderr            io.Writer
        trace             bool
        tracePolicyEnabled bool
        baseDir string
}

func NewRuntime(file string, stdout, stderr io.Writer, trace bool) *Runtime {
        rt := &Runtime{
                file:   file,
                gov:    newGovernanceState(),
                stdout: stdout,
                stderr: stderr,
                trace:  trace,
        }
        rt.global = newEnv(nil)
        rt.baseDir = filepath.Dir(file)
        rt.registerStdlib()
        return rt
}

func (rt *Runtime) traceLog(action string, line, col int) {
        if rt.trace {
                fmt.Fprintf(rt.stderr, "[TSR] file=%s line=%d regime=%s commit_depth=%d action=%s\n",
                        rt.file, line, rt.gov.Regime, rt.gov.CommitDepth, action)
        }
}

func (rt *Runtime) tracePolicyLog(msg string) {
        if rt.tracePolicyEnabled {
                fmt.Fprintf(rt.stderr, "[policy] -> %s\n", msg)
        }
}

func (rt *Runtime) Run(prog *Program) error {
        return rt.execStmts(prog.Stmts, rt.global)
}

func (rt *Runtime) execStmts(stmts []Stmt, env *Env) error {
        for _, s := range stmts {
                if err := rt.execStmt(s, env); err != nil {
                        return err
                }
        }
        return nil
}

func (rt *Runtime) execStmt(s Stmt, env *Env) error {
        switch st := s.(type) {
        case *LetStmt:
                val, err := rt.evalExpr(st.Value, env)
                if err != nil {
                        return err
                }
                env.set(st.Name, val)

        case *AssignStmt:
                val, err := rt.evalExpr(st.Value, env)
                if err != nil {
                        return err
                }
                if !env.assign(st.Name, val) {
                        return fmt.Errorf("%s:%d:%d: undefined variable %q", rt.file, st.Line, st.Col, st.Name)
                }

        case *SayStmt:
                val, err := rt.evalExpr(st.Value, env)
                if err != nil {
                        return err
                }
                fmt.Fprintln(rt.stdout, valueToString(val))

        case *IfStmt:
                if err := rt.gov.incrementBranch(rt.file, st.Line, st.Col); err != nil {
                        return err
                }
                cond, err := rt.evalExpr(st.Condition, env)
                if err != nil {
                        return err
                }
                if err := rt.gov.enterNest(rt.file, st.Line, st.Col); err != nil {
                        return err
                }
                if isTruthy(cond) {
                        childEnv := newEnv(env)
                        if err := rt.execStmts(st.Consequence, childEnv); err != nil {
                                rt.gov.exitNest()
                                return err
                        }
                } else if st.Alternative != nil {
                        childEnv := newEnv(env)
                        if err := rt.execStmts(st.Alternative, childEnv); err != nil {
                                rt.gov.exitNest()
                                return err
                        }
                }
                rt.gov.exitNest()

        case *WhileStmt:
                for {
                        if err := rt.gov.incrementLoop(rt.file, st.Line, st.Col); err != nil {
                                return err
                        }
                        cond, err := rt.evalExpr(st.Condition, env)
                        if err != nil {
                                return err
                        }
                        if !isTruthy(cond) {
                                break
                        }
                        if err := rt.gov.enterNest(rt.file, st.Line, st.Col); err != nil {
                                return err
                        }
                        childEnv := newEnv(env)
                        err = rt.execStmts(st.Body, childEnv)
                        rt.gov.exitNest()
                        if err != nil {
                                return err
                        }
                }

        case *FnStmt:
                fn := &Function{
                        Name:    st.Name,
                        Params:  st.Params,
                        Body:    st.Body,
                        Closure: env,
                }
                env.set(st.Name, &Value{Type: ValFunc, FuncVal: fn})

        case *ReturnStmt:
                val, err := rt.evalExpr(st.Value, env)
                if err != nil {
                        return err
                }
                return &ReturnSignal{Value: val}

        case *ImportStmt:
                if err := validatePath(st.Path); err != nil {
                        return fmt.Errorf("%s:%d:%d: %s", rt.file, st.Line, st.Col, err.Error())
                }
                modPath := filepath.Join(rt.baseDir, st.Path)
                modVal, err := rt.loadModule(modPath)
                if err != nil {
                        return fmt.Errorf("%s:%d:%d: import error: %s", rt.file, st.Line, st.Col, err.Error())
                }
                env.set(st.Alias, modVal)

        case *RegimeStmt:
                rt.gov.Regime = st.Mode
                rt.tracePolicyLog(fmt.Sprintf("regime=%s", st.Mode))

        case *StrainStmt:
                rt.gov.Budget = StrainBudget{
                        MaxBranch:    st.MaxBranch,
                        MaxLoop:      st.MaxLoop,
                        MaxCallDepth: st.MaxCallDepth,
                        MaxCommit:    st.MaxCommit,
                        MaxNest:      st.MaxNest,
                }
                rt.gov.BudgetSet = true
                rt.tracePolicyLog(fmt.Sprintf("strain={max_branch:%d, max_loop:%d, max_call_depth:%d, max_commit:%d, max_nest:%d}",
                        st.MaxBranch, st.MaxLoop, st.MaxCallDepth, st.MaxCommit, st.MaxNest))

        case *CapabilityStmt:
                rt.gov.addCapability(st.Kind, st.Pattern)
                rt.tracePolicyLog(fmt.Sprintf("capability %s %q", st.Kind, st.Pattern))

        case *CommitStmt:
                if err := rt.gov.enterCommit(rt.file, st.Line, st.Col); err != nil {
                        return err
                }
                if err := rt.gov.enterNest(rt.file, st.Line, st.Col); err != nil {
                        rt.gov.exitCommit()
                        return err
                }
                rt.traceLog("commit:"+st.Label, st.Line, st.Col)
                childEnv := newEnv(env)
                err := rt.execStmts(st.Body, childEnv)
                rt.gov.exitNest()
                rt.gov.exitCommit()
                if err != nil {
                        return err
                }

        case *ExprStmt:
                _, err := rt.evalExpr(st.Expr, env)
                return err
        }
        return nil
}

func (rt *Runtime) evalExpr(e Expr, env *Env) (*Value, error) {
        switch ex := e.(type) {
        case *NumberLit:
                return numVal(new(big.Rat).Set(ex.Value)), nil

        case *StringLit:
                return strVal(ex.Value), nil

        case *BoolLit:
                return boolVal(ex.Value), nil

        case *NilLit:
                return nilValue, nil

        case *Identifier:
                v, ok := env.get(ex.Name)
                if !ok {
                        return nil, fmt.Errorf("%s:%d:%d: undefined variable %q", rt.file, ex.Line, ex.Col, ex.Name)
                }
                return v, nil

        case *UnaryExpr:
                right, err := rt.evalExpr(ex.Right, env)
                if err != nil {
                        return nil, err
                }
                switch ex.Op {
                case "-":
                        if right.Type != ValNumber {
                                return nil, fmt.Errorf("%s:%d:%d: unary '-' requires number", rt.file, ex.Line, ex.Col)
                        }
                        r := new(big.Rat).Neg(right.NumVal)
                        return numVal(r), nil
                case "!":
                        return boolVal(!isTruthy(right)), nil
                }

        case *BinaryExpr:
                return rt.evalBinary(ex, env)

        case *ListLit:
                var elems []*Value
                for _, elem := range ex.Elements {
                        v, err := rt.evalExpr(elem, env)
                        if err != nil {
                                return nil, err
                        }
                        elems = append(elems, v)
                }
                return &Value{Type: ValList, ListVal: elems}, nil

        case *DictLit:
                d := make(map[string]*Value)
                for i, key := range ex.Keys {
                        v, err := rt.evalExpr(ex.Values[i], env)
                        if err != nil {
                                return nil, err
                        }
                        d[key] = v
                }
                return &Value{Type: ValDict, DictVal: d}, nil

        case *IndexExpr:
                obj, err := rt.evalExpr(ex.Object, env)
                if err != nil {
                        return nil, err
                }
                idx, err := rt.evalExpr(ex.Index, env)
                if err != nil {
                        return nil, err
                }
                return rt.evalIndex(obj, idx, ex.Line, ex.Col)

        case *DotExpr:
                obj, err := rt.evalExpr(ex.Object, env)
                if err != nil {
                        return nil, err
                }
                if obj.Type == ValModule {
                        v, ok := obj.ModVal.Members[ex.Field]
                        if !ok {
                                return nil, fmt.Errorf("%s:%d:%d: module has no member %q", rt.file, ex.Line, ex.Col, ex.Field)
                        }
                        return v, nil
                }
                return nil, fmt.Errorf("%s:%d:%d: dot access on non-module value", rt.file, ex.Line, ex.Col)

        case *CallExpr:
                callee, err := rt.evalExpr(ex.Callee, env)
                if err != nil {
                        return nil, err
                }
                var args []*Value
                for _, arg := range ex.Args {
                        v, err := rt.evalExpr(arg, env)
                        if err != nil {
                                return nil, err
                        }
                        args = append(args, v)
                }
                return rt.callFunc(callee, args, ex.Line, ex.Col)
        }
        return nilValue, nil
}

func (rt *Runtime) evalBinary(ex *BinaryExpr, env *Env) (*Value, error) {
        // Short-circuit for 'and' and 'or'
        if ex.Op == "and" {
                left, err := rt.evalExpr(ex.Left, env)
                if err != nil {
                        return nil, err
                }
                if !isTruthy(left) {
                        return falseValue, nil
                }
                right, err := rt.evalExpr(ex.Right, env)
                if err != nil {
                        return nil, err
                }
                return boolVal(isTruthy(right)), nil
        }
        if ex.Op == "or" {
                left, err := rt.evalExpr(ex.Left, env)
                if err != nil {
                        return nil, err
                }
                if isTruthy(left) {
                        return trueValue, nil
                }
                right, err := rt.evalExpr(ex.Right, env)
                if err != nil {
                        return nil, err
                }
                return boolVal(isTruthy(right)), nil
        }

        left, err := rt.evalExpr(ex.Left, env)
        if err != nil {
                return nil, err
        }
        right, err := rt.evalExpr(ex.Right, env)
        if err != nil {
                return nil, err
        }

        switch ex.Op {
        case "+":
                if left.Type == ValString && right.Type == ValString {
                        return strVal(left.StrVal + right.StrVal), nil
                }
                if left.Type == ValNumber && right.Type == ValNumber {
                        r := new(big.Rat).Add(left.NumVal, right.NumVal)
                        return numVal(r), nil
                }
                return nil, fmt.Errorf("%s:%d:%d: '+' requires two numbers or two strings", rt.file, ex.Line, ex.Col)
        case "-":
                if left.Type != ValNumber || right.Type != ValNumber {
                        return nil, fmt.Errorf("%s:%d:%d: '-' requires numbers", rt.file, ex.Line, ex.Col)
                }
                return numVal(new(big.Rat).Sub(left.NumVal, right.NumVal)), nil
        case "*":
                if left.Type != ValNumber || right.Type != ValNumber {
                        return nil, fmt.Errorf("%s:%d:%d: '*' requires numbers", rt.file, ex.Line, ex.Col)
                }
                return numVal(new(big.Rat).Mul(left.NumVal, right.NumVal)), nil
        case "/":
                if left.Type != ValNumber || right.Type != ValNumber {
                        return nil, fmt.Errorf("%s:%d:%d: '/' requires numbers", rt.file, ex.Line, ex.Col)
                }
                if right.NumVal.Sign() == 0 {
                        return nil, fmt.Errorf("%s:%d:%d: division by zero", rt.file, ex.Line, ex.Col)
                }
                return numVal(new(big.Rat).Quo(left.NumVal, right.NumVal)), nil
        case "==":
                return boolVal(valEqual(left, right)), nil
        case "!=":
                return boolVal(!valEqual(left, right)), nil
        case "<":
                c, err := numCmp(left, right, rt.file, ex.Line, ex.Col)
                if err != nil {
                        return nil, err
                }
                return boolVal(c < 0), nil
        case ">":
                c, err := numCmp(left, right, rt.file, ex.Line, ex.Col)
                if err != nil {
                        return nil, err
                }
                return boolVal(c > 0), nil
        case "<=":
                c, err := numCmp(left, right, rt.file, ex.Line, ex.Col)
                if err != nil {
                        return nil, err
                }
                return boolVal(c <= 0), nil
        case ">=":
                c, err := numCmp(left, right, rt.file, ex.Line, ex.Col)
                if err != nil {
                        return nil, err
                }
                return boolVal(c >= 0), nil
        }
        return nil, fmt.Errorf("%s:%d:%d: unknown operator %q", rt.file, ex.Line, ex.Col, ex.Op)
}

func (rt *Runtime) evalIndex(obj, idx *Value, line, col int) (*Value, error) {
        switch obj.Type {
        case ValList:
                if idx.Type != ValNumber {
                        return nil, fmt.Errorf("%s:%d:%d: list index must be number", rt.file, line, col)
                }
                if !idx.NumVal.IsInt() {
                        return nil, fmt.Errorf("%s:%d:%d: list index must be integer", rt.file, line, col)
                }
                i := int(idx.NumVal.Num().Int64())
                if i < 0 {
                        i = len(obj.ListVal) + i
                }
                if i < 0 || i >= len(obj.ListVal) {
                        return nil, fmt.Errorf("%s:%d:%d: list index %d out of bounds (len=%d)", rt.file, line, col, i, len(obj.ListVal))
                }
                return obj.ListVal[i], nil
        case ValDict:
                if idx.Type != ValString {
                        return nil, fmt.Errorf("%s:%d:%d: dict key must be string", rt.file, line, col)
                }
                v, ok := obj.DictVal[idx.StrVal]
                if !ok {
                        return nilValue, nil
                }
                return v, nil
        default:
                return nil, fmt.Errorf("%s:%d:%d: cannot index %s", rt.file, line, col, valueTypeName(obj))
        }
}

func (rt *Runtime) callFunc(callee *Value, args []*Value, line, col int) (*Value, error) {
        if callee.Type != ValFunc {
                return nil, fmt.Errorf("%s:%d:%d: cannot call non-function %s", rt.file, line, col, valueTypeName(callee))
        }
        fn := callee.FuncVal

        if fn.Builtin != nil {
                return fn.Builtin(args, rt.gov, rt)
        }

        if len(args) != len(fn.Params) {
                return nil, fmt.Errorf("%s:%d:%d: function %q expects %d args, got %d", rt.file, line, col, fn.Name, len(fn.Params), len(args))
        }

        if err := rt.gov.enterCall(rt.file, line, col); err != nil {
                return nil, err
        }
        defer rt.gov.exitCall()

        if err := rt.gov.enterNest(rt.file, line, col); err != nil {
                return nil, err
        }
        defer rt.gov.exitNest()

        fnEnv := newEnv(fn.Closure)
        for i, param := range fn.Params {
                fnEnv.set(param, args[i])
        }

        err := rt.execStmts(fn.Body, fnEnv)
        if err != nil {
                if ret, ok := err.(*ReturnSignal); ok {
                        return ret.Value, nil
                }
                return nil, err
        }
        return nilValue, nil
}

func (rt *Runtime) loadModule(path string) (*Value, error) {
        src, err := os.ReadFile(path)
        if err != nil {
                return nil, fmt.Errorf("cannot read module %q: %s", path, err)
        }
        modRT := &Runtime{
                file:    path,
                gov:     newGovernanceState(),
                stdout:  rt.stdout,
                stderr:  rt.stderr,
                trace:   rt.trace,
                baseDir: filepath.Dir(path),
        }
        modRT.global = newEnv(nil)
        modRT.registerStdlib()

        lexer := NewLexer(string(src), path)
        tokens, err := lexer.Tokenize()
        if err != nil {
                return nil, err
        }
        parser := NewParser(tokens, path)
        prog, err := parser.Parse()
        if err != nil {
                return nil, err
        }
        if err := modRT.Run(prog); err != nil {
                return nil, err
        }

        // Export top-level names
        mod := &Module{
                Name:    filepath.Base(path),
                Members: make(map[string]*Value),
        }
        for name, val := range modRT.global.vars {
                mod.Members[name] = val
        }
        return &Value{Type: ValModule, ModVal: mod}, nil
}

// ---- helpers ----

func isTruthy(v *Value) bool {
        switch v.Type {
        case ValNil:
                return false
        case ValBool:
                return v.BoolVal
        case ValNumber:
                return v.NumVal.Sign() != 0
        case ValString:
                return v.StrVal != ""
        default:
                return true
        }
}

func valEqual(a, b *Value) bool {
        if a.Type != b.Type {
                return false
        }
        switch a.Type {
        case ValNil:
                return true
        case ValBool:
                return a.BoolVal == b.BoolVal
        case ValNumber:
                return a.NumVal.Cmp(b.NumVal) == 0
        case ValString:
                return a.StrVal == b.StrVal
        default:
                return false
        }
}

func numCmp(a, b *Value, file string, line, col int) (int, error) {
        if a.Type != ValNumber || b.Type != ValNumber {
                return 0, fmt.Errorf("%s:%d:%d: comparison requires numbers", file, line, col)
        }
        return a.NumVal.Cmp(b.NumVal), nil
}

func valueTypeName(v *Value) string {
        switch v.Type {
        case ValNil:
                return "nil"
        case ValBool:
                return "bool"
        case ValNumber:
                return "number"
        case ValString:
                return "string"
        case ValList:
                return "list"
        case ValDict:
                return "dict"
        case ValFunc:
                return "function"
        case ValModule:
                return "module"
        }
        return "unknown"
}

func valueToString(v *Value) string {
        switch v.Type {
        case ValNil:
                return "nil"
        case ValBool:
                if v.BoolVal {
                        return "true"
                }
                return "false"
        case ValNumber:
                if v.NumVal.IsInt() {
                        return v.NumVal.Num().String()
                }
                f, _ := v.NumVal.Float64()
                return fmt.Sprintf("%g", f)
        case ValString:
                return v.StrVal
        case ValList:
                var parts []string
                for _, e := range v.ListVal {
                        parts = append(parts, valueToString(e))
                }
                return "[" + strings.Join(parts, ", ") + "]"
        case ValDict:
                var parts []string
                for k, val := range v.DictVal {
                        parts = append(parts, fmt.Sprintf("%q: %s", k, valueToString(val)))
                }
                return "{" + strings.Join(parts, ", ") + "}"
        case ValFunc:
                return "<function:" + v.FuncVal.Name + ">"
        case ValModule:
                return "<module:" + v.ModVal.Name + ">"
        }
        return "nil"
}

// RunFile parses and runs a .tsr file. Returns exit code, stdout, stderr.
func RunFile(path string, trace, tracePolicy bool) (int, string, string) {
        src, err := os.ReadFile(path)
        if err != nil {
                return 1, "", fmt.Sprintf("error: cannot read %q: %s\n", path, err)
        }

        var outBuf, errBuf strings.Builder
        lexer := NewLexer(string(src), path)
        tokens, err := lexer.Tokenize()
        if err != nil {
                return 1, "", err.Error() + "\n"
        }
        parser := NewParser(tokens, path)
        prog, err := parser.Parse()
        if err != nil {
                return 1, "", err.Error() + "\n"
        }

        rt := NewRuntime(path, &outBuf, &errBuf, trace)
        rt.tracePolicyEnabled = tracePolicy
        if tracePolicy {
                fmt.Fprintf(&errBuf, "[policy] source=tsr:%s\n", path)
        }
        if err := rt.Run(prog); err != nil {
                return 1, outBuf.String(), errBuf.String() + err.Error() + "\n"
        }
        return 0, outBuf.String(), errBuf.String()
}

// RunSource runs TSR source code from a string (for testing)
func RunSource(source, filename string, trace bool) (int, string, string) {
        var outBuf, errBuf strings.Builder
        lexer := NewLexer(source, filename)
        tokens, err := lexer.Tokenize()
        if err != nil {
                return 1, "", err.Error() + "\n"
        }
        parser := NewParser(tokens, filename)
        prog, err := parser.Parse()
        if err != nil {
                return 1, "", err.Error() + "\n"
        }

        rt := NewRuntime(filename, &outBuf, &errBuf, trace)
        if err := rt.Run(prog); err != nil {
                return 1, outBuf.String(), errBuf.String() + err.Error() + "\n"
        }
        return 0, outBuf.String(), errBuf.String()
}
