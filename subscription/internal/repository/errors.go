package repository

import "errors"

var (
	ErrDuplicate = errors.New("duplicate")
	ErrNotFound  = errors.New("not found")
	ErrUnknown   = errors.New("unknown")
	ErrInvalid   = errors.New("invalid")
)
