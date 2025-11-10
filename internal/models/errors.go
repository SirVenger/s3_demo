package models

import "errors"

var (
	ErrNotFound   = errors.New("file not found")
	ErrIncomplete = errors.New("file incomplete")
	ErrNoStorage  = errors.New("no storage ready")
)
