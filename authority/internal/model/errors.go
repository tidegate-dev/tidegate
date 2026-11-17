package model

import "errors"

// ErrChangedSinceRead means the data changed after it was read. Nothing was
// written.
var ErrChangedSinceRead = errors.New("the data changed after it was read")
