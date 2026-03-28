package db

import "errors"

// ErrNotInitialized is returned when database operations are attempted
// before Init() has been called successfully.
var ErrNotInitialized = errors.New("database not initialized")

// ErrDBNotAvailable is returned when the database connection cannot be
// established (e.g., SQLite binary not available).
var ErrDBNotAvailable = errors.New("database not available")
