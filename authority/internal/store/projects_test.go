//go:build integration

package store

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

var _ = Describe("ProjectReadWriter", func() {
	var data *Store

	BeforeEach(func(ctx SpecContext) {
		_, data = newStore(ctx)
	})

	It("stores a project at Commit", func(ctx SpecContext) {
		project := &model.Project{ID: model.ProjectID{1}, Name: "payments"}

		unit := data.NewUnit()
		unit.Projects().Add(project)
		Expect(data.Projects().All(ctx)).To(BeEmpty())

		Expect(unit.Commit(ctx)).To(Succeed())
		Expect(data.Projects().All(ctx)).To(ConsistOf(project))
	})

	It("returns ErrProjectNameTaken at Commit for a name that differs only in letter case", func(ctx SpecContext) {
		addProject(ctx, data, model.ProjectID{1}, "payments")

		unit := data.NewUnit()
		unit.Projects().Add(&model.Project{ID: model.ProjectID{2}, Name: "Payments"})

		Expect(unit.Commit(ctx)).To(MatchError(model.ErrProjectNameTaken))
	})

	It("returns the projects ordered by name, ignoring letter case", func(ctx SpecContext) {
		addProject(ctx, data, model.ProjectID{1}, "search")
		addProject(ctx, data, model.ProjectID{2}, "Payments")
		addProject(ctx, data, model.ProjectID{3}, "audit")

		projects, err := data.Projects().All(ctx)

		Expect(err).NotTo(HaveOccurred())
		Expect(projects).To(HaveExactElements(
			HaveField("Name", "audit"),
			HaveField("Name", "Payments"),
			HaveField("Name", "search"),
		))
	})
})

func addProject(ctx SpecContext, data *Store, id model.ProjectID, name string) {
	GinkgoHelper()

	unit := data.NewUnit()
	unit.Projects().Add(&model.Project{ID: id, Name: name})
	Expect(unit.Commit(ctx)).To(Succeed())
}
