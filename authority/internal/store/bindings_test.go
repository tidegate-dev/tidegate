//go:build integration

package store

import (
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

var _ = Describe("BindingReadWriter", func() {
	var (
		pool  *pgxpool.Pool
		data  *Store
		alice *model.User
	)

	BeforeEach(func(ctx SpecContext) {
		pool, data = newStore(ctx)
		addProject(ctx, data, model.ProjectID{1}, "payments")
		addRole(ctx, data, model.RoleID{2}, model.ProjectID{1}, "operator")
		alice = addUser(ctx, data, "alice")
	})

	Describe("Add", func() {
		It("stores a binding at Commit, which FindByID then returns", func(ctx SpecContext) {
			binding := &model.Binding{
				ID:        model.BindingID{3},
				ProjectID: model.ProjectID{1},
				RoleID:    model.RoleID{2},
				UserID:    alice.ID,
			}

			unit := data.NewUnit()
			unit.Bindings().Add(binding)
			Expect(count(ctx, pool, "bindings")).To(Equal(0))

			Expect(unit.Commit(ctx)).To(Succeed())
			Expect(data.Bindings().FindByID(ctx, binding.ID)).To(Equal(binding))
		})

		It("returns ErrBindingExists at Commit for a role that the user already has in the project", func(ctx SpecContext) {
			addBinding(ctx, data, model.BindingID{3}, model.ProjectID{1}, model.RoleID{2}, alice.ID)

			unit := data.NewUnit()
			unit.Bindings().Add(&model.Binding{
				ID:        model.BindingID{4},
				ProjectID: model.ProjectID{1},
				RoleID:    model.RoleID{2},
				UserID:    alice.ID,
			})

			Expect(unit.Commit(ctx)).To(MatchError(model.ErrBindingExists))
			Expect(count(ctx, pool, "bindings")).To(Equal(1))
		})

		It("returns ErrRoleNotFound at Commit for a role of another project", func(ctx SpecContext) {
			addProject(ctx, data, model.ProjectID{4}, "search")
			addRole(ctx, data, model.RoleID{5}, model.ProjectID{4}, "operator")

			unit := data.NewUnit()
			unit.Bindings().Add(&model.Binding{
				ID:        model.BindingID{3},
				ProjectID: model.ProjectID{1},
				RoleID:    model.RoleID{5},
				UserID:    alice.ID,
			})

			Expect(unit.Commit(ctx)).To(MatchError(model.ErrRoleNotFound))
			Expect(count(ctx, pool, "bindings")).To(Equal(0))
		})

		It("returns ErrRoleNotFound at Commit for a role that doesn't exist", func(ctx SpecContext) {
			unit := data.NewUnit()
			unit.Bindings().Add(&model.Binding{
				ID:        model.BindingID{3},
				ProjectID: model.ProjectID{1},
				RoleID:    model.RoleID{9},
				UserID:    alice.ID,
			})

			Expect(unit.Commit(ctx)).To(MatchError(model.ErrRoleNotFound))
		})

		It("returns ErrUserNotFound at Commit for a user that doesn't exist", func(ctx SpecContext) {
			unit := data.NewUnit()
			unit.Bindings().Add(&model.Binding{
				ID:        model.BindingID{3},
				ProjectID: model.ProjectID{1},
				RoleID:    model.RoleID{2},
				UserID:    model.UserID{9},
			})

			Expect(unit.Commit(ctx)).To(MatchError(model.ErrUserNotFound))
		})
	})

	Describe("Remove", func() {
		It("deletes the binding at Commit", func(ctx SpecContext) {
			binding := addBinding(ctx, data, model.BindingID{3}, model.ProjectID{1}, model.RoleID{2}, alice.ID)

			unit := data.NewUnit()
			unit.Bindings().Remove(binding)
			Expect(count(ctx, pool, "bindings")).To(Equal(1))

			Expect(unit.Commit(ctx)).To(Succeed())
			Expect(count(ctx, pool, "bindings")).To(Equal(0))
		})

		It("leaves the other bindings", func(ctx SpecContext) {
			bob := addUser(ctx, data, "bob")
			binding := addBinding(ctx, data, model.BindingID{3}, model.ProjectID{1}, model.RoleID{2}, alice.ID)
			other := addBinding(ctx, data, model.BindingID{4}, model.ProjectID{1}, model.RoleID{2}, bob.ID)

			unit := data.NewUnit()
			unit.Bindings().Remove(binding)

			Expect(unit.Commit(ctx)).To(Succeed())
			Expect(data.Bindings().FindByID(ctx, other.ID)).To(Equal(other))
		})

		It("succeeds for a binding that is already gone", func(ctx SpecContext) {
			unit := data.NewUnit()
			unit.Bindings().Remove(&model.Binding{ID: model.BindingID{3}})

			Expect(unit.Commit(ctx)).To(Succeed())
		})
	})

	It("loses the bindings of a user who is removed", func(ctx SpecContext) {
		addBinding(ctx, data, model.BindingID{3}, model.ProjectID{1}, model.RoleID{2}, alice.ID)

		unit := data.NewUnit()
		unit.Users().Remove(alice)

		Expect(unit.Commit(ctx)).To(Succeed())
		Expect(count(ctx, pool, "bindings")).To(Equal(0))
	})
})

var _ = Describe("BindingReader", func() {
	var data *Store

	BeforeEach(func(ctx SpecContext) {
		_, data = newStore(ctx)
		addProject(ctx, data, model.ProjectID{1}, "payments")
	})

	Describe("FindByID", func() {
		It("returns ErrBindingNotFound for an ID that no binding has", func(ctx SpecContext) {
			_, err := data.Bindings().FindByID(ctx, model.BindingID{3})

			Expect(err).To(MatchError(model.ErrBindingNotFound))
		})
	})

	Describe("ListForProjectID", func() {
		It("returns the bindings of the project with the names of their role and user", func(ctx SpecContext) {
			addRole(ctx, data, model.RoleID{2}, model.ProjectID{1}, "operator")
			alice := addUser(ctx, data, "alice")
			addBinding(ctx, data, model.BindingID{3}, model.ProjectID{1}, model.RoleID{2}, alice.ID)

			Expect(data.Bindings().ListForProjectID(ctx, model.ProjectID{1})).To(Equal([]*model.BindingWithNames{
				{ID: model.BindingID{3}, ProjectID: model.ProjectID{1}, RoleName: "operator", UserName: "alice"},
			}))
		})

		It("leaves out the bindings of other projects", func(ctx SpecContext) {
			addProject(ctx, data, model.ProjectID{4}, "search")
			addRole(ctx, data, model.RoleID{5}, model.ProjectID{4}, "operator")
			alice := addUser(ctx, data, "alice")
			addBinding(ctx, data, model.BindingID{6}, model.ProjectID{4}, model.RoleID{5}, alice.ID)

			Expect(data.Bindings().ListForProjectID(ctx, model.ProjectID{1})).To(BeEmpty())
		})

		It("orders by the role name and then the user name, ignoring letter case", func(ctx SpecContext) {
			addRole(ctx, data, model.RoleID{2}, model.ProjectID{1}, "operator")
			addRole(ctx, data, model.RoleID{3}, model.ProjectID{1}, "Auditor")
			alice := addUser(ctx, data, "alice")
			bob := addUser(ctx, data, "Bob")
			addBinding(ctx, data, model.BindingID{4}, model.ProjectID{1}, model.RoleID{2}, bob.ID)
			addBinding(ctx, data, model.BindingID{5}, model.ProjectID{1}, model.RoleID{2}, alice.ID)
			addBinding(ctx, data, model.BindingID{6}, model.ProjectID{1}, model.RoleID{3}, bob.ID)

			Expect(data.Bindings().ListForProjectID(ctx, model.ProjectID{1})).To(Equal([]*model.BindingWithNames{
				{ID: model.BindingID{6}, ProjectID: model.ProjectID{1}, RoleName: "Auditor", UserName: "Bob"},
				{ID: model.BindingID{5}, ProjectID: model.ProjectID{1}, RoleName: "operator", UserName: "alice"},
				{ID: model.BindingID{4}, ProjectID: model.ProjectID{1}, RoleName: "operator", UserName: "Bob"},
			}))
		})
	})
})

func addBinding(
	ctx SpecContext,
	data *Store,
	id model.BindingID,
	projectID model.ProjectID,
	roleID model.RoleID,
	userID model.UserID,
) *model.Binding {
	GinkgoHelper()

	binding := &model.Binding{ID: id, ProjectID: projectID, RoleID: roleID, UserID: userID}

	unit := data.NewUnit()
	unit.Bindings().Add(binding)
	Expect(unit.Commit(ctx)).To(Succeed())

	return binding
}
