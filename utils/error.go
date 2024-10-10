package utils

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// Error is an error with an attached stacktrace.
type Error struct {
	Err   error
	Stack []uintptr
}

// NewError makes an Error from the given error. The stacktrace
// will point to the line of code that called NewError.
func NewError(err error) *Error {
	const MAX_STACK_DEPTH = 50
	stack := make([]uintptr, MAX_STACK_DEPTH)
	length := runtime.Callers(2, stack[:])

	return &Error{
		Err:   err,
		Stack: stack[:length],
	}
}

// Error returns a string that contains both the
// error message and the callstack.
func (err *Error) Error() string {
	var b strings.Builder
	b.WriteString(reflect.TypeOf(err.Err).String())
	b.WriteByte(' ')
	b.WriteString(err.Err.Error())
	b.WriteByte('\n')
	frames := runtime.CallersFrames(err.Stack)
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&b, "\t%s:%d (0x%x)\n", frame.File, frame.Line, frame.PC)
		if !more {
			break
		}
	}
	return b.String()
}
