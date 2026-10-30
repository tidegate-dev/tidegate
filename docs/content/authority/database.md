# Database

The authority keeps its data in Postgres. Point it to a database with a connection URL:

```sh
TIDEGATE_AUTHORITY__DATABASE__URL='postgres://tidegate:tidegate@db.example.com:5432/tidegate?sslmode=verify-full' \
authority
```

Set `sslmode=verify-full`, so the authority only connects to a Postgres server with a valid certificate. Without an `sslmode`, the connection may fall back to plain text, and the authority logs a warning when it starts.

Before you start a new version of the authority, run `authority migrate` with the same variables. It brings the database schema up to date and exits. The authority refuses to start if the schema is older than it expects.

With `TIDEGATE_AUTHORITY__DATABASE__MIGRATE_ON_START` set to `true`, the authority migrates when it starts instead. Its database user then needs permission to change the schema. Without that, the user needs to read and write the authority's tables, and `SELECT` on the table `goose_db_version`, where the authority reads the schema version.
