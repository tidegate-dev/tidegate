//go:build integration

package store

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

var _ = Describe("DummyReadWriter", func() {
	var data *Store

	BeforeEach(func(ctx SpecContext) {
		_, data = newStore(ctx)
	})

	It("gives a stored dummy its ID at Commit", func(ctx SpecContext) {
		dummy := &model.Dummy{Text: "hello"}

		unit := data.NewUnit()
		unit.Dummies().Add(dummy)
		Expect(dummy.ID).To(BeZero())

		Expect(unit.Commit(ctx)).To(Succeed())
		Expect(data.Dummies().FindByID(ctx, dummy.ID)).To(Equal(dummy))
	})

	It("returns ErrDummyNotFound for an ID that no dummy has", func(ctx SpecContext) {
		_, err := data.Dummies().FindByID(ctx, 1)

		Expect(err).To(MatchError(model.ErrDummyNotFound))
	})
})
