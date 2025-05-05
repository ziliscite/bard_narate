package repository

import "errors"

var (
	ErrNilPool   = errors.New("database pool is nil")
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("already exists")
	ErrUnknown   = errors.New("unknown error")
	ErrInvalid   = errors.New("invalid")
)
