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

	It("returns a stored text by its ID", func(ctx SpecContext) {
		id, err := data.Dummies().Add(ctx, "hello")
		Expect(err).NotTo(HaveOccurred())

		Expect(data.Dummies().ByID(ctx, id)).To(Equal("hello"))
	})

	It("returns ErrDummyNotFound for an ID that no text has", func(ctx SpecContext) {
		_, err := data.Dummies().ByID(ctx, 1)

		Expect(err).To(MatchError(model.ErrDummyNotFound))
	})
})
