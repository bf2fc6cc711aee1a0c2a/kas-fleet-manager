package errors

import "fmt"

var _ error = &UndefinedVariableError{}

// UndefinedVariableError is an error for when expected variables are undefined (this can include zero-values for types
// such as strings).
type UndefinedVariableError struct {
	// variableName is the name of the undefined variable this error is referencing.
	variableName string
}

func NewUndefinedVariableError(variableName string) error {
	return &UndefinedVariableError{
		variableName: variableName,
	}
}

func (u UndefinedVariableError) Error() string {
	return fmt.Sprintf("variable '%s' is not defined", u.variableName)
}
