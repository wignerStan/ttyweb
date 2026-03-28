package db

import "errors"

var (
	// ErrNotInitialized is returned when DB operations are attempted before Init().
	ErrNotInitialized = errors.New("database not initialized: call db.Init() first")
	// ErrDBNotAvailable is returned when the database is intentionally disabled.
	ErrDBNotAvailable = errors.New("database is not available")
)
