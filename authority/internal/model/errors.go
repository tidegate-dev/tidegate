package model

import "errors"

// ErrConflict means a row changed after it was read, or another row already
// uses a value that must be unique. Nothing was written.
var ErrConflict = errors.New("the data changed in the meantime, read it again and retry")
