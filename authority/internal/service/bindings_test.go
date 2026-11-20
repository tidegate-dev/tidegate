//go:build integration

package service

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/storetest"
)

var _ = Describe("Bindings", func() {
	var (
		units    model.UnitFactory
		projects *Projects
		roles    *Roles
		bindings *Bindings
	)

	BeforeEach(func(ctx SpecContext) {
		_, data := storetest.New(ctx)
		units = data
		projects = NewProjects(data.Projects(), data)
		roles = NewRoles(data.Projects(), data)
		bindings = NewBindings(data.Projects(), data.Roles(), data.Users(), data.Bindings(), data)
	})

	Describe("Create", func() {
		It("returns the new binding with the stored names of its role and user, which List then returns too", func(ctx SpecContext) {
			project, err := projects.Create(ctx, "payments")
			Expect(err).NotTo(HaveOccurred())
			_, err = roles.Create(ctx, "payments", "operator", model.ProjectPermissions{})
			Expect(err).NotTo(HaveOccurred())
			storetest.AddUser(ctx, units, "alice")

			binding, err := bindings.Create(ctx, "payments", "operator", "Alice")

			Expect(err).NotTo(HaveOccurred())
			Expect(binding.ID).NotTo(BeZero())
			Expect(binding.ProjectID).To(Equal(project.ID))
			Expect(binding.RoleName).To(Equal("operator"))
			Expect(binding.UserName).To(Equal("alice"))
			Expect(bindings.List(ctx, "payments")).To(ConsistOf(binding))
		})

		It("returns ErrProjectNotFound for a project that doesn't exist", func(ctx SpecContext) {
			storetest.AddUser(ctx, units, "alice")

			_, err := bindings.Create(ctx, "payments", "operator", "alice")

			Expect(err).To(MatchError(model.ErrProjectNotFound))
		})

		It("returns ErrRoleNotFound for a role that the project doesn't have", func(ctx SpecContext) {
			_, err := projects.Create(ctx, "payments")
			Expect(err).NotTo(HaveOccurred())
			storetest.AddUser(ctx, units, "alice")

			_, err = bindings.Create(ctx, "payments", "operator", "alice")

			Expect(err).To(MatchError(model.ErrRoleNotFound))
		})

		DescribeTable("returns ErrUserNotFound",
			func(ctx SpecContext, userName string) {
				_, err := projects.Create(ctx, "payments")
				Expect(err).NotTo(HaveOccurred())
				_, err = roles.Create(ctx, "payments", "operator", model.ProjectPermissions{})
				Expect(err).NotTo(HaveOccurred())
				storetest.AddUser(ctx, units, "alice")

				_, err = bindings.Create(ctx, "payments", "operator", userName)

				Expect(err).To(MatchError(model.ErrUserNotFound))
			},
			Entry("for a user that doesn't exist", "bob"),
			Entry("for a name with NUL, which Postgres refuses in a query", "ali\x00ce"),
		)
	})

	Describe("List", func() {
		It("returns ErrProjectNotFound for a project that doesn't exist", func(ctx SpecContext) {
			_, err := bindings.List(ctx, "payments")

			Expect(err).To(MatchError(model.ErrProjectNotFound))
		})
	})
})
