//go:build integration

package service

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/phemera-dev/phemera/authority/internal/storetest"
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
	})
})
