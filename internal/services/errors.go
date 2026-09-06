package services

import "errors"

var (
	ErrAlreadyReturned        = errors.New("already returned")
	ErrReturnQuantityExceeded = errors.New("return quantity exceeds sold quantity")
	ErrInvalidReturnState     = errors.New("invalid return state")
)
