//go:build integration

package store

import (
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

var _ = Describe("TicketReadWriter", func() {
	var (
		pool   *pgxpool.Pool
		unit   model.Unit
		ticket model.Ticket
	)

	BeforeEach(func(ctx SpecContext) {
		dbPool, data := newStore(ctx)
		pool = dbPool
		projectID := model.ProjectID{1}
		addProject(ctx, data, projectID, "payments")
		alice := addUser(ctx, data, "alice")
		asset := newSSHAsset(model.AssetID{4}, projectID, "host")
		addAsset(ctx, data, asset)

		ticket = model.Ticket{ID: model.TicketID{5}, AssetID: asset.ID, UserID: alice.ID}
		unit = data.NewUnit()
		unit.Tickets().Add(&ticket)
	})

	It("stores the asset and user for the ticket", func(ctx SpecContext) {
		Expect(unit.Commit(ctx)).To(Succeed())

		var assetID, userID string

		row := pool.QueryRow(ctx, "SELECT asset_id::text, user_id::text FROM tickets WHERE id = $1", ticket.ID.String())
		Expect(row.Scan(&assetID, &userID)).To(Succeed())
		Expect(assetID).To(Equal(ticket.AssetID.String()))
		Expect(userID).To(Equal(ticket.UserID.String()))
	})

	It("sets the ticket version from the inserted row", func(ctx SpecContext) {
		Expect(unit.Commit(ctx)).To(Succeed())

		var xmin pgtype.Uint32

		Expect(pool.QueryRow(ctx, "SELECT xmin FROM tickets WHERE id = $1", ticket.ID.String()).Scan(&xmin)).To(Succeed())
		Expect(xmin.Valid).To(BeTrue())
		Expect(ticket.Version).To(Equal(xmin.Uint32))
	})

	It("rejects a ticket if its asset disappears before commit", func(ctx SpecContext) {
		_, err := pool.Exec(ctx, "DELETE FROM assets WHERE id = $1", ticket.AssetID.String())
		Expect(err).NotTo(HaveOccurred())
		Expect(unit.Commit(ctx)).To(MatchError(model.ErrAssetNotFound))
		Expect(count(ctx, pool, "tickets")).To(Equal(0))
	})

	It("rejects a ticket if its user disappears before commit", func(ctx SpecContext) {
		_, err := pool.Exec(ctx, "DELETE FROM users WHERE id = $1", ticket.UserID.String())
		Expect(err).NotTo(HaveOccurred())
		Expect(unit.Commit(ctx)).To(MatchError(model.ErrUserNotFound))
		Expect(count(ctx, pool, "tickets")).To(Equal(0))
	})
})
