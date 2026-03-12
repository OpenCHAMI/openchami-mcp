package errorsx

import "fmt"

type PermissionError struct {
	Message string
}

func (e *PermissionError) Error() string {
	return e.Message
}

type InvalidArgumentError struct {
	Message string
}

func (e *InvalidArgumentError) Error() string {
	return e.Message
}

type NotImplementedError struct {
	Message string
}

func (e *NotImplementedError) Error() string {
	return e.Message
}

func NewPermission(format string, args ...any) error {
	return &PermissionError{Message: fmt.Sprintf(format, args...)}
}

func NewInvalidArgument(format string, args ...any) error {
	return &InvalidArgumentError{Message: fmt.Sprintf(format, args...)}
}

func NewNotImplemented(format string, args ...any) error {
	return &NotImplementedError{Message: fmt.Sprintf(format, args...)}
}
