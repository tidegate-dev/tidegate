//go:build integration

package store

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tidegate-dev/tidegate/authority/internal/db"
	"github.com/tidegate-dev/tidegate/authority/internal/dbtest"
	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "store integration")
}

// newStore returns a store on an empty, migrated database, and the pool the
// store uses.
func newStore(ctx context.Context) (*pgxpool.Pool, *Store) {
	GinkgoHelper()

	pool, err := db.Open(ctx, dbtest.NewDatabase())
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(pool.Close)
	Expect(db.Migrate(ctx, pool)).To(Succeed())

	return pool, New(pool)
}

func newUser(name string) *model.User {
	GinkgoHelper()

	id, err := model.NewUserID()
	Expect(err).NotTo(HaveOccurred())

	return &model.User{ID: id, Name: name}
}

// addUser stores a user without a password.
func addUser(ctx context.Context, data *Store, name string) *model.User {
	GinkgoHelper()

	user := newUser(name)

	unit := data.NewUnit()
	unit.Users().Add(user)
	Expect(unit.Commit(ctx)).To(Succeed())

	return user
}

func newSession(userID model.UserID, token string) *model.Session {
	GinkgoHelper()

	id, err := model.NewSessionID()
	Expect(err).NotTo(HaveOccurred())

	return &model.Session{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash(token),
		CreatedAt: issuedAt,
		ExpiresAt: issuedAt.Add(sessionLifetime),
	}
}

// addSession stores a session that lasts sessionLifetime from issuedAt.
func addSession(ctx context.Context, data *Store, userID model.UserID, token string) *model.Session {
	GinkgoHelper()

	session := newSession(userID, token)

	unit := data.NewUnit()
	unit.Sessions().Add(session)
	Expect(unit.Commit(ctx)).To(Succeed())

	return session
}

// tokenHash returns 32 bytes that start with token, because the sessions
// table refuses a hash of another length.
func tokenHash(token string) secret.Bytes {
	hash := make([]byte, 32)
	copy(hash, token)

	return secret.NewBytes(hash)
}

func count(ctx context.Context, pool *pgxpool.Pool, table string) int {
	GinkgoHelper()

	var rows int

	Expect(pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&rows)).To(Succeed())

	return rows
}
