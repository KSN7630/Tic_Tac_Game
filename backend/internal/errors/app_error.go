package errors

import "fmt"

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
