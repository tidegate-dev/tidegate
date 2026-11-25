//go:build integration

package store

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/phemera-dev/phemera/authority/internal/model"
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
		Expect(data.Projects().ListAll(ctx)).To(BeEmpty())

		Expect(unit.Commit(ctx)).To(Succeed())
		Expect(data.Projects().ListAll(ctx)).To(ConsistOf(project))
	})

	It("returns ErrProjectNameTaken at Commit for a name that differs only in letter case", func(ctx SpecContext) {
		addProject(ctx, data, model.ProjectID{1}, "payments")

		unit := data.NewUnit()
		unit.Projects().Add(&model.Project{ID: model.ProjectID{2}, Name: "Payments"})

		Expect(unit.Commit(ctx)).To(MatchError(model.ErrProjectNameTaken))
	})

	It("doesn't call the name taken at Commit when another project has the ID", func(ctx SpecContext) {
		addProject(ctx, data, model.ProjectID{1}, "payments")

		unit := data.NewUnit()
		unit.Projects().Add(&model.Project{ID: model.ProjectID{1}, Name: "search"})

		err := unit.Commit(ctx)

		Expect(err).To(HaveOccurred())
		Expect(err).NotTo(MatchError(model.ErrProjectNameTaken))
	})

	It("returns the project with the name, ignoring letter case", func(ctx SpecContext) {
		addProject(ctx, data, model.ProjectID{1}, "payments")
		addProject(ctx, data, model.ProjectID{2}, "search")

		project, err := data.Projects().FindByName(ctx, "Payments")

		Expect(err).NotTo(HaveOccurred())
		Expect(project).To(Equal(&model.Project{ID: model.ProjectID{1}, Name: "payments"}))
	})

	It("returns ErrProjectNotFound for a name that no project has", func(ctx SpecContext) {
		addProject(ctx, data, model.ProjectID{1}, "payments")

		_, err := data.Projects().FindByName(ctx, "search")

		Expect(err).To(MatchError(model.ErrProjectNotFound))
	})

	It("returns the projects ordered by name, ignoring letter case", func(ctx SpecContext) {
		addProject(ctx, data, model.ProjectID{1}, "search")
		addProject(ctx, data, model.ProjectID{2}, "Payments")
		addProject(ctx, data, model.ProjectID{3}, "audit")

		projects, err := data.Projects().ListAll(ctx)

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
