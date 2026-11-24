package secretengine

import (
	"errors"
	"fmt"
	"maps"
)

// ErrUnknownEngine means that no engine has the name that was asked for.
var ErrUnknownEngine = errors.New("unknown secret engine")

// Engines holds the engines that are set up, each under its name.
type Engines struct {
	enginesByName map[string]Engine
}

// NewEngines returns the Engines for a map from each engine's name to the
// engine. It copies the map.
func NewEngines(enginesByName map[string]Engine) *Engines {
	return &Engines{enginesByName: maps.Clone(enginesByName)}
}

// ByName returns the engine with the name. If there is none, the error wraps
// ErrUnknownEngine.
func (e *Engines) ByName(engineName string) (Engine, error) {
	engine, ok := e.enginesByName[engineName]
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrUnknownEngine, engineName)
	}

	return engine, nil
}
