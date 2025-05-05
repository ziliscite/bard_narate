package repository

import "fmt"

var (
	ErrNotExist  = fmt.Errorf("does not exist")
	ErrDuplicate = fmt.Errorf("already exists")
)
