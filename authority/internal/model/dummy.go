package model

import "errors"

// ErrDummyNotFound means no dummy has the given ID.
var ErrDummyNotFound = errors.New("dummy not found")
