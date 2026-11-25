//go:build integration

package service

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/phemera-dev/phemera/authority/internal/model"
	"github.com/phemera-dev/phemera/authority/internal/storetest"
)

var _ = Describe("Roles", func() {
	var (
		projects *Projects
		roles    *Roles
	)

	BeforeEach(func(ctx SpecContext) {
		_, data := storetest.New(ctx)
		projects = NewProjects(data.Projects(), data)
		roles = NewRoles(data.Projects(), data)
	})

	Describe("Create", func() {
		It("returns the new role in the project with the name", func(ctx SpecContext) {
			project, err := projects.Create(ctx, "payments")
			Expect(err).NotTo(HaveOccurred())

			permissions := model.ProjectPermissions{CreateRoles: true, Bind: true}

			role, err := roles.Create(ctx, "payments", "operator", permissions)

			Expect(err).NotTo(HaveOccurred())
			Expect(role.ID).NotTo(BeZero())
			Expect(role.ProjectID).To(Equal(project.ID))
			Expect(role.Name).To(Equal("operator"))
			Expect(role.Permissions).To(Equal(permissions))
		})

		It("returns ErrRoleNameTaken for a name that a role of the project has", func(ctx SpecContext) {
			_, err := projects.Create(ctx, "payments")
			Expect(err).NotTo(HaveOccurred())
			_, err = roles.Create(ctx, "payments", "operator", model.ProjectPermissions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = roles.Create(ctx, "payments", "operator", model.ProjectPermissions{})

			Expect(err).To(MatchError(model.ErrRoleNameTaken))
		})

		It("returns ErrProjectNotFound for a project that doesn't exist", func(ctx SpecContext) {
			_, err := roles.Create(ctx, "payments", "operator", model.ProjectPermissions{})

			Expect(err).To(MatchError(model.ErrProjectNotFound))
		})
	})
})
