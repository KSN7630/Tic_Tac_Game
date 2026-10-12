package errors

import (
	"errors"
	"fmt"
)

type ErrorCode string

type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func (e AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func New(code ErrorCode, msg string) AppError {
	return AppError{Code: code, Message: msg}
}

func As(err error) (AppError, bool) {
	var ae AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return AppError{}, false
}
