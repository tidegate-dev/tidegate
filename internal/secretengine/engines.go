package secretengine

import (
	"errors"
	"fmt"
)

// ErrUnknownEngine means that no engine has the name that was asked for.
var ErrUnknownEngine = errors.New("unknown secret engine")

// Engines holds the engines that are set up, each under its name.
type Engines struct {
	byName map[string]Engine
}

// NewEngines returns the Engines for a map from each engine's name to the
// engine.
func NewEngines(byName map[string]Engine) *Engines {
	return &Engines{byName: byName}
}

// ByName returns the engine with the name. If there is none, the error wraps
// ErrUnknownEngine.
func (e *Engines) ByName(engineName string) (Engine, error) {
	engine, ok := e.byName[engineName]
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrUnknownEngine, engineName)
	}

	return engine, nil
}
