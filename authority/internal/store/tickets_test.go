//go:build integration

package store

import (
	"github.com/jackc/pgx/v5/pgtype"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

var _ = Describe("TicketReadWriter", func() {
	It("stores the ticket and its row version", func(ctx SpecContext) {
		pool, data := newStore(ctx)
		projectID := model.ProjectID{1}

		addProject(ctx, data, projectID, "payments")
		alice := addUser(ctx, data, "alice")
		asset := newSSHAsset(model.AssetID{4}, projectID, "host")
		addAsset(ctx, data, asset)

		id, err := model.NewTicketID()
		Expect(err).NotTo(HaveOccurred())

		ticket := &model.Ticket{ID: id, AssetID: asset.ID, UserID: alice.ID}
		unit := data.NewUnit()
		unit.Tickets().Add(ticket)
		Expect(unit.Commit(ctx)).To(Succeed())

		var xmin pgtype.Uint32

		var storedAsset, storedUser string
		Expect(pool.QueryRow(ctx, "SELECT xmin, asset_id::text, user_id::text FROM tickets WHERE id = $1", id.String()).Scan(&xmin, &storedAsset, &storedUser)).To(Succeed())
		Expect(storedAsset).To(Equal(asset.ID.String()))
		Expect(storedUser).To(Equal(alice.ID.String()))
		Expect(xmin.Valid).To(BeTrue())
		Expect(ticket.Version).To(Equal(xmin.Uint32))
	})
})
