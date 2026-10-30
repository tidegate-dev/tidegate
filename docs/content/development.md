# Development environment

The [README](https://github.com/tidegate-dev/tidegate#development) lists the tools you need and how to get them with Nix.

Once you have them, run `just` in the repository to see the recipes. `just ci` runs every check that CI runs.

To work on this site, run `just docs::serve` and open the address it prints. The page reloads when you change a file in `docs/content/`.

`arch-go.yml` says which packages may import which. `internal/` imports nothing from a component or from `gen/`, and a component imports only itself, `internal/` and the generated code in `gen/`. Every package needs a rule there. If you add a new top-level directory, like a new component, add a rule for it too. `just arch` checks the rules, and `just ci` and the pre-commit hook run it. Note that arch-go reads packages like `go build` does, so it doesn't check imports in test files or in files behind a build tag.

The API is defined in `.proto` files in `proto/`. After you change one, run `just generate` and commit the generated code in `gen/`. `just ci` fails if the generated code doesn't match the `.proto` files.

The authority's SQL queries are in `authority/internal/db/queries/`, and sqlc generates Go code for them in `authority/internal/db/sqlc/`. After you change a query, also run `just generate` and commit the result.

To change the database schema, add a migration to `authority/internal/db/migrations/`, numbered after the last one. A migration only goes forward and only adds to the schema. To remove a table or a column, wait a release after the code stops using it, so an old and a new authority can share the database during an update. `just dev` and `just kind` migrate when the authority starts.

## Run tidegate locally

```sh
just dev
```

This starts tidegate's components and the services they need, like Postgres, with [process-compose](https://f1bonacc1.github.io/process-compose/). The components run from your working copy. After you change one, restart it in process-compose, which builds it again.

To debug a component, stop it in process-compose and start it in your editor's debugger. To run all components yourself, start only the services:

```sh
just dev-deps
```

The services keep their data between runs. To start with empty data, stop `just dev` or `just dev-deps` and run `just dev-reset`. `dev/tidegate.yaml` defines the components and `dev/deps.yaml` the services, with their ports and logins.

The authority's integration tests run against Postgres. Start the services first, then run the tests:

```sh
just dev-deps
just test-integration
```

`just ci` runs them too. Each test creates its own database and drops it at the end. If you stop a test run before it finishes, its databases stay behind. On the Postgres from `just dev-deps`, `just dev-reset` deletes them. To use another Postgres server, set `TIDEGATE_TEST_DATABASE_URL` to its URL. Its user needs the `CREATEDB` privilege.

Integration tests are [Ginkgo](https://onsi.github.io/ginkgo/) specs with [Gomega](https://onsi.github.io/gomega/) assertions. Their files start with `//go:build integration`, so `just test` and `go test ./...` leave them out. Unit tests use the standard `testing` package with testify. To create a database for a spec, call `dbtest.NewDatabase` from `authority/internal/dbtest`.

One of the services is an SSH host to try tidegate against. Connect to it with the password `tidegate`:

```sh
ssh -p 2222 tidegate@127.0.0.1
```

Its host key is committed in `dev/ssh-host/`, so the host has the same fingerprint for everyone.

To run tidegate in Kubernetes instead, start a local cluster with [kind](https://kind.sigs.k8s.io/):

```sh
just kind
```

This builds the components' container images, creates the cluster if it doesn't exist yet, installs cert-manager with Helm and deploys everything in `dev/kind/`. Run it again after a change. Note that kind also switches your kubectl context to the new cluster. `just kind-down` deletes the cluster.

In both environments, the authority serves HTTPS on port 8080 with a certificate from a local CA. `just dev` and `just kind` create the CA in `dev/tls/` the first time. In `just kind`, cert-manager issues the authority's certificate from the same CA. To trust the CA, point `SSL_CERT_FILE` to it:

```sh
SSL_CERT_FILE=dev/tls/ca.pem tidegate ping
```

Other tools take the CA file as a flag, like `buf curl --cacert dev/tls/ca.pem`. Your system's trust store stays unchanged.
