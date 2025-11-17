package models

import "errors"

var (
	// ErrNotFound возвращается, когда метаданные файла отсутствуют в хранилище.
	ErrNotFound = errors.New("file not found")
	// ErrIncomplete сигнализирует о пропавших частях файла.
	ErrIncomplete = errors.New("file incomplete")
	// ErrNoStorage означает отсутствие живых стораджей для распределения частей.
	ErrNoStorage = errors.New("no storage ready")
)
