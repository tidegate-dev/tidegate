//go:build integration

package service

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/storetest"
)

var _ = Describe("Projects", func() {
	var projects *Projects

	BeforeEach(func(ctx SpecContext) {
		_, data := storetest.New(ctx)
		projects = NewProjects(data.Projects(), data)
	})

	Describe("Create", func() {
		It("returns the new project, which List then returns too", func(ctx SpecContext) {
			project, err := projects.Create(ctx, "payments")

			Expect(err).NotTo(HaveOccurred())
			Expect(project.Name).To(Equal("payments"))
			Expect(project.ID).NotTo(BeZero())
			Expect(projects.List(ctx)).To(ConsistOf(project))
		})

		It("refuses a name that another project has", func(ctx SpecContext) {
			_, err := projects.Create(ctx, "payments")
			Expect(err).NotTo(HaveOccurred())

			_, err = projects.Create(ctx, "payments")

			Expect(err).To(MatchError(model.ErrProjectNameTaken))
			Expect(projects.List(ctx)).To(HaveLen(1))
		})

		DescribeTable("refuses an invalid name",
			func(ctx SpecContext, name string) {
				_, err := projects.Create(ctx, name)

				Expect(err).To(MatchError(ErrInvalidName))
				Expect(projects.List(ctx)).To(BeEmpty())
			},
			Entry("empty", ""),
			Entry("leading space", " payments"),
			Entry("NUL", "pay\x00ments"),
		)
	})
})
