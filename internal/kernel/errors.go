package kernel

import "fmt"

type KernelError struct {
	File    string
	Line    int
	Col     int
	Message string
}

func (e *KernelError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("%s:%d:%d: kernel error: %s", e.File, e.Line, e.Col, e.Message)
	}
	return fmt.Sprintf("%d:%d: kernel error: %s", e.Line, e.Col, e.Message)
}

func newKernelError(file string, line, col int, msg string, args ...interface{}) *KernelError {
	return &KernelError{
		File:    file,
		Line:    line,
		Col:     col,
		Message: fmt.Sprintf(msg, args...),
	}
}
