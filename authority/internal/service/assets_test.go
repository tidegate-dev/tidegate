//go:build integration

package service

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/storetest"
)

var _ = Describe("Assets", func() {
	var (
		projects *Projects
		assets   *Assets
		target   model.SSHTarget
	)

	BeforeEach(func(ctx SpecContext) {
		_, data := storetest.New(ctx)
		projects = NewProjects(data.Projects(), data)
		assets = NewAssets(data.Projects(), data.Assets(), data)
		target = model.SSHTarget{Host: "bastion.example", Port: 22, HostKey: "ssh-ed25519 AAAA"}
	})

	Describe("CreateSSH", func() {
		It("returns the new SSH asset of the project, which List then returns too", func(ctx SpecContext) {
			project, err := projects.Create(ctx, "payments")
			Expect(err).NotTo(HaveOccurred())

			asset, err := assets.CreateSSH(ctx, "payments", "bastion", map[string]string{"env": "prod"}, target)

			Expect(err).NotTo(HaveOccurred())
			Expect(asset.ID).NotTo(BeZero())
			Expect(asset.ProjectID).To(Equal(project.ID))
			Expect(asset.Name).To(Equal("bastion"))
			Expect(asset.Kind).To(Equal(model.AssetKindSSH))
			Expect(asset.Labels).To(Equal(map[string]string{"env": "prod"}))
			Expect(asset.SSH).To(Equal(&target))
			Expect(assets.List(ctx, "payments")).To(ConsistOf(asset))
		})

		It("returns ErrProjectNotFound for a project that doesn't exist", func(ctx SpecContext) {
			_, err := assets.CreateSSH(ctx, "payments", "bastion", nil, target)

			Expect(err).To(MatchError(model.ErrProjectNotFound))
		})
	})

	Describe("List", func() {
		It("returns ErrProjectNotFound for a project that doesn't exist", func(ctx SpecContext) {
			_, err := assets.List(ctx, "payments")

			Expect(err).To(MatchError(model.ErrProjectNotFound))
		})
	})
})
