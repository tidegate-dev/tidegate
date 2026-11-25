//go:build integration

package postgres_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/secretengine/postgres"
	"github.com/tidegate-dev/tidegate/internal/secret"
	"github.com/tidegate-dev/tidegate/internal/secretengine"
)

var _ = Describe("Engine", func() {
	Describe("Fetch", func() {
		It("returns the value that was stored under the reference", func(ctx SpecContext) {
			engine := newEngine(ctx)
			reference, err := engine.Store(ctx, secret.NewBytes([]byte("a stored value")))
			Expect(err).NotTo(HaveOccurred())

			value, err := engine.Fetch(ctx, reference)

			Expect(err).NotTo(HaveOccurred())
			Expect(value.Reveal()).To(Equal([]byte("a stored value")))
		})

		It("returns ErrNotFound for a reference that has no value", func(ctx SpecContext) {
			engine := newEngine(ctx)

			_, err := engine.Fetch(ctx, "5b0e7c1a-3f52-4d0c-9a41-6f2d8c7e9b13")

			Expect(err).To(MatchError(secretengine.ErrNotFound))
		})

		It("returns ErrNotFound for a reference that isn't a UUID", func(ctx SpecContext) {
			engine := newEngine(ctx)

			_, err := engine.Fetch(ctx, "not a UUID")

			Expect(err).To(MatchError(secretengine.ErrNotFound))
		})
	})

	Describe("Store", func() {
		It("returns a different reference each time it stores the same value", func(ctx SpecContext) {
			engine := newEngine(ctx)

			first, err := engine.Store(ctx, secret.NewBytes([]byte("a stored value")))
			Expect(err).NotTo(HaveOccurred())
			second, err := engine.Store(ctx, secret.NewBytes([]byte("a stored value")))
			Expect(err).NotTo(HaveOccurred())

			Expect(second).NotTo(Equal(first))
		})
	})

	Describe("Delete", func() {
		It("removes the value that was stored under the reference", func(ctx SpecContext) {
			engine := newEngine(ctx)
			reference, err := engine.Store(ctx, secret.NewBytes([]byte("a stored value")))
			Expect(err).NotTo(HaveOccurred())

			Expect(engine.Delete(ctx, reference)).To(Succeed())

			_, err = engine.Fetch(ctx, reference)
			Expect(err).To(MatchError(secretengine.ErrNotFound))
		})

		It("returns ErrNotFound for a reference that has no value", func(ctx SpecContext) {
			engine := newEngine(ctx)

			err := engine.Delete(ctx, "5b0e7c1a-3f52-4d0c-9a41-6f2d8c7e9b13")

			Expect(err).To(MatchError(secretengine.ErrNotFound))
		})

		It("returns ErrNotFound for a reference that isn't a UUID", func(ctx SpecContext) {
			engine := newEngine(ctx)

			err := engine.Delete(ctx, "not a UUID")

			Expect(err).To(MatchError(secretengine.ErrNotFound))
		})
	})
})

func newEngine(ctx context.Context) secretengine.Engine {
	GinkgoHelper()

	pool := newPool(ctx)
	Expect(postgres.CreateSchema(ctx, pool)).To(Succeed())

	return postgres.New(pool)
}
