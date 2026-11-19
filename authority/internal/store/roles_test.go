//go:build integration

package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
)

var _ = Describe("RoleReadWriter", func() {
	var (
		pool *pgxpool.Pool
		data *Store
	)

	BeforeEach(func(ctx SpecContext) {
		pool, data = newStore(ctx)
	})

	DescribeTable("stores a role at Commit",
		func(ctx SpecContext, permissions model.ProjectPermissions) {
			addProject(ctx, data, model.ProjectID{1}, "payments")

			role := &model.Role{
				ID:          model.RoleID{2},
				ProjectID:   model.ProjectID{1},
				Name:        "operator",
				Permissions: permissions,
			}

			unit := data.NewUnit()
			unit.Roles().Add(role)
			Expect(count(ctx, pool, "roles")).To(Equal(0))

			Expect(unit.Commit(ctx)).To(Succeed())
			Expect(storedRole(ctx, pool, role.ID)).To(Equal(role))
		},
		Entry("that may create roles", model.ProjectPermissions{CreateRoles: true}),
		Entry("that may escalate", model.ProjectPermissions{Escalate: true}),
		Entry("that may create bindings", model.ProjectPermissions{CreateBindings: true}),
		Entry("that may bind", model.ProjectPermissions{Bind: true}),
	)

	It("returns ErrRoleNameTaken at Commit for a name that differs only in letter case in the same project", func(ctx SpecContext) {
		addProject(ctx, data, model.ProjectID{1}, "payments")
		addRole(ctx, data, model.RoleID{2}, model.ProjectID{1}, "operator")

		unit := data.NewUnit()
		unit.Roles().Add(&model.Role{ID: model.RoleID{3}, ProjectID: model.ProjectID{1}, Name: "Operator"})

		Expect(unit.Commit(ctx)).To(MatchError(model.ErrRoleNameTaken))
	})

	It("stores a role with a name that a role of another project has", func(ctx SpecContext) {
		addProject(ctx, data, model.ProjectID{1}, "payments")
		addProject(ctx, data, model.ProjectID{2}, "search")
		addRole(ctx, data, model.RoleID{3}, model.ProjectID{1}, "operator")

		unit := data.NewUnit()
		unit.Roles().Add(&model.Role{ID: model.RoleID{4}, ProjectID: model.ProjectID{2}, Name: "operator"})

		Expect(unit.Commit(ctx)).To(Succeed())
	})

	It("doesn't call the name taken at Commit when another role has the ID", func(ctx SpecContext) {
		addProject(ctx, data, model.ProjectID{1}, "payments")
		addRole(ctx, data, model.RoleID{2}, model.ProjectID{1}, "operator")

		unit := data.NewUnit()
		unit.Roles().Add(&model.Role{ID: model.RoleID{2}, ProjectID: model.ProjectID{1}, Name: "auditor"})

		err := unit.Commit(ctx)

		Expect(err).To(HaveOccurred())
		Expect(err).NotTo(MatchError(model.ErrRoleNameTaken))
	})

	It("returns ErrProjectNotFound at Commit for a project that doesn't exist", func(ctx SpecContext) {
		unit := data.NewUnit()
		unit.Roles().Add(&model.Role{ID: model.RoleID{1}, ProjectID: model.ProjectID{2}, Name: "operator"})

		Expect(unit.Commit(ctx)).To(MatchError(model.ErrProjectNotFound))
	})
})

func addRole(ctx SpecContext, data *Store, id model.RoleID, projectID model.ProjectID, name string) {
	GinkgoHelper()

	unit := data.NewUnit()
	unit.Roles().Add(&model.Role{ID: id, ProjectID: projectID, Name: name})
	Expect(unit.Commit(ctx)).To(Succeed())
}

// storedRole reads the row with SQL, because the store has no reader for roles.
func storedRole(ctx context.Context, pool *pgxpool.Pool, id model.RoleID) *model.Role {
	GinkgoHelper()

	var projectID pgtype.UUID

	role := &model.Role{ID: id}

	Expect(pool.QueryRow(ctx,
		"SELECT project_id, name, create_roles, escalate, create_bindings, bind FROM roles WHERE id = $1",
		databaseID(id),
	).Scan(
		&projectID,
		&role.Name,
		&role.Permissions.CreateRoles,
		&role.Permissions.Escalate,
		&role.Permissions.CreateBindings,
		&role.Permissions.Bind,
	)).To(Succeed())

	role.ProjectID = model.ProjectID(projectID.Bytes)

	return role
}
