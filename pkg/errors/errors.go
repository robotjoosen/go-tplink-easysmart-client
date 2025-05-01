package errors

import "errors"

var (
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrInvalidResponse      = errors.New("invalid response")
)
