package service

import (
	"context"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

// DummyFinder finds a dummy by its ID.
type DummyFinder interface {
	// FindByID returns the dummy with the ID, or model.ErrDummyNotFound.
	FindByID(ctx context.Context, id int64) (*model.Dummy, error)
}

// Dummies checks the path from a client into the database and back.
type Dummies struct {
	dummies DummyFinder
	units   model.UnitFactory
}

// NewDummies returns Dummies that read from dummies and write in units of
// work from units.
func NewDummies(dummies DummyFinder, units model.UnitFactory) *Dummies {
	return &Dummies{dummies: dummies, units: units}
}

// Write stores text and returns its ID.
func (d *Dummies) Write(ctx context.Context, text string) (int64, error) {
	dummy := &model.Dummy{Text: text}

	unit := d.units.NewUnit()
	unit.Dummies().Add(dummy)

	err := unit.Commit(ctx)
	if err != nil {
		return 0, err
	}

	return dummy.ID, nil
}

// Read returns the text with the given ID, or model.ErrDummyNotFound.
func (d *Dummies) Read(ctx context.Context, id int64) (string, error) {
	dummy, err := d.dummies.FindByID(ctx, id)
	if err != nil {
		return "", err
	}

	return dummy.Text, nil
}
