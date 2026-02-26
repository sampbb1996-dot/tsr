package tsr

import "fmt"

type TSRError struct {
        File    string
        Line    int
        Col     int
        Message string
}

func (e *TSRError) Error() string {
        if e.File != "" {
                return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Col, e.Message)
        }
        return fmt.Sprintf("%d:%d: %s", e.Line, e.Col, e.Message)
}

func newError(file string, line, col int, msg string, args ...interface{}) *TSRError {
        return &TSRError{
                File:    file,
                Line:    line,
                Col:     col,
                Message: fmt.Sprintf(msg, args...),
        }
}

// ReturnSignal is used for return statements (not an error)
type ReturnSignal struct {
        Value *Value
}

func (r *ReturnSignal) Error() string {
        return "return"
}
