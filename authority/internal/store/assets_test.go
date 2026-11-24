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
		Expect(details).To(MatchJSON(`{
			"host": "bastion.example",
			"port": 2222,
			"host_key": "ssh-ed25519 AAAA",
			"user_name": "deploy",
			"secret": {"engine_name": "tidegate", "reference": "reference-1", "kind": "private_key"}
		}`))
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
		func(ctx SpecContext, change func(asset *model.Asset), message string) {
			asset := newSSHAsset(model.AssetID{2}, model.ProjectID{1}, "bastion")
			change(asset)

			unit := data.NewUnit()
			unit.Assets().Add(asset)

			Expect(unit.Commit(ctx)).To(MatchError(ContainSubstring(message)))
			Expect(count(ctx, pool, "assets")).To(Equal(0))
		},
		Entry("without a target", func(asset *model.Asset) { asset.SSH = nil },
			"an SSH asset needs a host, a port, a host key and a user name"),
		Entry("without a host", func(asset *model.Asset) { asset.SSH.Host = "" },
			"an SSH asset needs a host, a port, a host key and a user name"),
		Entry("without a port", func(asset *model.Asset) { asset.SSH.Port = 0 },
			"an SSH asset needs a host, a port, a host key and a user name"),
		Entry("without a host key", func(asset *model.Asset) { asset.SSH.HostKey = "" },
			"an SSH asset needs a host, a port, a host key and a user name"),
		Entry("without a user name", func(asset *model.Asset) { asset.SSH.UserName = "" },
			"an SSH asset needs a host, a port, a host key and a user name"),
		Entry("without a stored secret", func(asset *model.Asset) { asset.SSH.Secret = model.StoredSSHSecret{} },
			"an SSH asset needs the engine name and the reference of its stored secret"),
		Entry("without the engine name of its stored secret", func(asset *model.Asset) { asset.SSH.Secret.EngineName = "" },
			"an SSH asset needs the engine name and the reference of its stored secret"),
		Entry("without the reference of its stored secret", func(asset *model.Asset) { asset.SSH.Secret.Reference = "" },
			"an SSH asset needs the engine name and the reference of its stored secret"),
		Entry("with a stored secret of an unknown kind", func(asset *model.Asset) { asset.SSH.Secret.Kind = "certificate" },
			`unknown SSH secret kind "certificate"`),
		Entry("with a stored secret without a kind", func(asset *model.Asset) { asset.SSH.Secret.Kind = "" },
			`unknown SSH secret kind ""`),
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

		Expect(err).To(MatchError(ContainSubstring("an SSH asset needs a host, a port, a host key and a user name")))
	})

	It("fails to read an SSH asset whose details lack the user name and the stored secret", func(ctx SpecContext) {
		_, err := pool.Exec(ctx,
			`INSERT INTO assets (id, project_id, name, kind, labels, details)
			 VALUES ($1, $2, 'bastion', 'ssh', '{}', '{"host": "bastion.example", "port": 22, "host_key": "ssh-ed25519 AAAA"}')`,
			databaseID(model.AssetID{2}), databaseID(model.ProjectID{1}),
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = data.Assets().ListForProjectID(ctx, model.ProjectID{1})

		Expect(err).To(MatchError(ContainSubstring("an SSH asset needs a host, a port, a host key and a user name")))
	})

	It("fails to read an SSH asset whose stored secret lacks the reference", func(ctx SpecContext) {
		_, err := pool.Exec(ctx,
			`INSERT INTO assets (id, project_id, name, kind, labels, details)
			 VALUES ($1, $2, 'bastion', 'ssh', '{}', '{
			     "host": "bastion.example", "port": 22, "host_key": "ssh-ed25519 AAAA",
			     "user_name": "deploy", "secret": {"engine_name": "tidegate", "kind": "password"}
			 }')`,
			databaseID(model.AssetID{2}), databaseID(model.ProjectID{1}),
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = data.Assets().ListForProjectID(ctx, model.ProjectID{1})

		Expect(err).To(MatchError(ContainSubstring("an SSH asset needs the engine name and the reference of its stored secret")))
	})
})

func newSSHAsset(id model.AssetID, projectID model.ProjectID, name string) *model.Asset {
	return &model.Asset{
		ID:        id,
		ProjectID: projectID,
		Name:      name,
		Kind:      model.AssetKindSSH,
		Labels:    map[string]string{},
		SSH: &model.SSHTarget{
			Host:     "bastion.example",
			Port:     2222,
			HostKey:  "ssh-ed25519 AAAA",
			UserName: "deploy",
			Secret: model.StoredSSHSecret{
				EngineName: "tidegate",
				Reference:  "reference-1",
				Kind:       model.SSHSecretKindPrivateKey,
			},
		},
	}
}

func addAsset(ctx SpecContext, data *Store, asset *model.Asset) {
	GinkgoHelper()

	unit := data.NewUnit()
	unit.Assets().Add(asset)
	Expect(unit.Commit(ctx)).To(Succeed())
}
