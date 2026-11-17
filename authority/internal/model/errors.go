package model

import "errors"

// ErrConflict means a row changed after it was read, or another row already
// uses a value that must be unique. Nothing was written.
var ErrConflict = errors.New("the write conflicts with the stored data")
