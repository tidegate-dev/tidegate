//go:build integration

package store

import (
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

var _ = Describe("AssetReadWriter", func() {
	var (
		pool *pgxpool.Pool
		data *Store
	)

	BeforeEach(func(ctx SpecContext) {
		pool, data = newStore(ctx)
		addProject(ctx, data, model.ProjectID{1}, "payments")
	})

	It("stores an SSH asset at Commit, which ListForProjectID then returns", func(ctx SpecContext) {
		asset := newSSHAsset(model.AssetID{2}, model.ProjectID{1}, "bastion")
		asset.Labels = map[string]string{"env": "prod", "team": "payments"}

		unit := data.NewUnit()
		unit.Assets().Add(asset)
		Expect(data.Assets().ListForProjectID(ctx, model.ProjectID{1})).To(BeEmpty())

		Expect(unit.Commit(ctx)).To(Succeed())
		Expect(data.Assets().ListForProjectID(ctx, model.ProjectID{1})).To(ConsistOf(asset))
	})

	It("writes the kind and the SSH target as JSON into the details column", func(ctx SpecContext) {
		addAsset(ctx, data, newSSHAsset(model.AssetID{2}, model.ProjectID{1}, "bastion"))

		var (
			kind    string
			details string
		)

		Expect(pool.QueryRow(ctx, "SELECT kind, details FROM assets").Scan(&kind, &details)).To(Succeed())
		Expect(kind).To(Equal("ssh"))
		Expect(details).To(MatchJSON(`{"host": "bastion.example", "port": 2222, "host_key": "ssh-ed25519 AAAA"}`))
	})

	It("returns an asset that was stored without labels with no labels", func(ctx SpecContext) {
		asset := newSSHAsset(model.AssetID{2}, model.ProjectID{1}, "bastion")
		asset.Labels = nil
		addAsset(ctx, data, asset)

		assets, err := data.Assets().ListForProjectID(ctx, model.ProjectID{1})

		Expect(err).NotTo(HaveOccurred())
		Expect(assets).To(ConsistOf(HaveField("Labels", BeEmpty())))
	})

	It("returns ErrAssetNameTaken at Commit for a name that differs only in letter case in the same project", func(ctx SpecContext) {
		addAsset(ctx, data, newSSHAsset(model.AssetID{2}, model.ProjectID{1}, "bastion"))

		unit := data.NewUnit()
		unit.Assets().Add(newSSHAsset(model.AssetID{3}, model.ProjectID{1}, "Bastion"))

		Expect(unit.Commit(ctx)).To(MatchError(model.ErrAssetNameTaken))
	})

	It("stores an asset with a name that an asset of another project has", func(ctx SpecContext) {
		addProject(ctx, data, model.ProjectID{2}, "search")
		addAsset(ctx, data, newSSHAsset(model.AssetID{3}, model.ProjectID{1}, "bastion"))

		unit := data.NewUnit()
		unit.Assets().Add(newSSHAsset(model.AssetID{4}, model.ProjectID{2}, "bastion"))

		Expect(unit.Commit(ctx)).To(Succeed())
	})

	It("doesn't call the name taken at Commit when another asset has the ID", func(ctx SpecContext) {
		addAsset(ctx, data, newSSHAsset(model.AssetID{2}, model.ProjectID{1}, "bastion"))

		unit := data.NewUnit()
		unit.Assets().Add(newSSHAsset(model.AssetID{2}, model.ProjectID{1}, "database"))

		err := unit.Commit(ctx)

		Expect(err).To(HaveOccurred())
		Expect(err).NotTo(MatchError(model.ErrAssetNameTaken))
	})

	It("returns ErrProjectNotFound at Commit for a project that doesn't exist", func(ctx SpecContext) {
		unit := data.NewUnit()
		unit.Assets().Add(newSSHAsset(model.AssetID{2}, model.ProjectID{9}, "bastion"))

		Expect(unit.Commit(ctx)).To(MatchError(model.ErrProjectNotFound))
	})

	DescribeTable("stores nothing at Commit for an SSH asset",
		func(ctx SpecContext, change func(asset *model.Asset)) {
			asset := newSSHAsset(model.AssetID{2}, model.ProjectID{1}, "bastion")
			change(asset)

			unit := data.NewUnit()
			unit.Assets().Add(asset)

			Expect(unit.Commit(ctx)).To(MatchError(ContainSubstring("an SSH asset needs a host, a port and a host key")))
			Expect(count(ctx, pool, "assets")).To(Equal(0))
		},
		Entry("without a target", func(asset *model.Asset) { asset.SSH = nil }),
		Entry("without a host", func(asset *model.Asset) { asset.SSH.Host = "" }),
		Entry("without a port", func(asset *model.Asset) { asset.SSH.Port = 0 }),
		Entry("without a host key", func(asset *model.Asset) { asset.SSH.HostKey = "" }),
	)

	It("stores nothing at Commit for an asset of an unknown kind", func(ctx SpecContext) {
		asset := newSSHAsset(model.AssetID{2}, model.ProjectID{1}, "bastion")
		asset.Kind = "database"

		unit := data.NewUnit()
		unit.Assets().Add(asset)

		Expect(unit.Commit(ctx)).To(MatchError(ContainSubstring(`unknown kind "database"`)))
		Expect(count(ctx, pool, "assets")).To(Equal(0))
	})

	It("returns only the assets of the project, ordered by name, ignoring letter case", func(ctx SpecContext) {
		addProject(ctx, data, model.ProjectID{2}, "search")
		addAsset(ctx, data, newSSHAsset(model.AssetID{3}, model.ProjectID{1}, "web"))
		addAsset(ctx, data, newSSHAsset(model.AssetID{4}, model.ProjectID{1}, "Bastion"))
		addAsset(ctx, data, newSSHAsset(model.AssetID{5}, model.ProjectID{1}, "database"))
		addAsset(ctx, data, newSSHAsset(model.AssetID{6}, model.ProjectID{2}, "index"))

		assets, err := data.Assets().ListForProjectID(ctx, model.ProjectID{1})

		Expect(err).NotTo(HaveOccurred())
		Expect(assets).To(HaveExactElements(
			HaveField("Name", "Bastion"),
			HaveField("Name", "database"),
			HaveField("Name", "web"),
		))
	})

	It("fails to read an SSH asset whose details lack the host key", func(ctx SpecContext) {
		_, err := pool.Exec(ctx,
			`INSERT INTO assets (id, project_id, name, kind, labels, details)
			 VALUES ($1, $2, 'bastion', 'ssh', '{}', '{"host": "bastion.example", "port": 22}')`,
			databaseID(model.AssetID{2}), databaseID(model.ProjectID{1}),
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = data.Assets().ListForProjectID(ctx, model.ProjectID{1})

		Expect(err).To(MatchError(ContainSubstring("an SSH asset needs a host, a port and a host key")))
	})
})

func newSSHAsset(id model.AssetID, projectID model.ProjectID, name string) *model.Asset {
	return &model.Asset{
		ID:        id,
		ProjectID: projectID,
		Name:      name,
		Kind:      model.AssetKindSSH,
		Labels:    map[string]string{},
		SSH:       &model.SSHTarget{Host: "bastion.example", Port: 2222, HostKey: "ssh-ed25519 AAAA"},
	}
}

func addAsset(ctx SpecContext, data *Store, asset *model.Asset) {
	GinkgoHelper()

	unit := data.NewUnit()
	unit.Assets().Add(asset)
	Expect(unit.Commit(ctx)).To(Succeed())
}
