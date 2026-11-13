package store

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tidegate-dev/tidegate/internal/secret"
)

func databaseID[ID ~[16]byte](id ID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func databaseVersion(version uint32) pgtype.Uint32 {
	return pgtype.Uint32{Uint32: version, Valid: true}
}

func databaseTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func databaseHash(hash *secret.String) pgtype.Text {
	if hash == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: hash.Reveal(), Valid: true}
}
