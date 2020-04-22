package helpers

import (
	"fmt"
	"runtime"
)

// ErrorNotImplemented - functionality not implemented
type ErrorNotImplemented struct {
	Caller string
}

// NewErrNotImplemented - create not implemented error
func NewErrNotImplemented() ErrorNotImplemented {
	var caller string
	var err error
	if caller, err = getCallerMethodName(); err != nil {
		caller = ""
	}
	return ErrorNotImplemented{Caller: caller}
}

func (e ErrorNotImplemented) Error() string {
	if len(e.Caller) > 0 {
		return fmt.Sprintf("not implemented (%s)", e.Caller)
	}
	return "not implemented (%s)"
}

func getCallerMethodName() (string, error) {
	fpcs := make([]uintptr, 1)
	// Skip 3 levels to get the caller
	n := runtime.Callers(3, fpcs)
	if n == 0 {
		return "", fmt.Errorf("no caller")
	}

	caller := runtime.FuncForPC(fpcs[0] - 1)
	if caller == nil {
		return "", fmt.Errorf("msg caller is nil")
	}

	return caller.Name(), nil
}
