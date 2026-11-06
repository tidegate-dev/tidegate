# Development environment

The [README](https://github.com/tidegate-dev/tidegate#development) lists the tools you need and how to get them with Nix.

Once you have them, run `just` in the repository to see the recipes. `just ci` runs every check that CI runs, except fuzzing and `just mutate`. Both are described below.

To work on this site, run `just docs::serve` and open the address it prints. The page reloads when you change a file in `docs/content/`.

`arch-go.yml` says which packages may import which. `internal/` imports nothing from a component or from `gen/`, and a component imports only itself, `internal/` and the generated code in `gen/`. Every package needs a rule there. If you add a new top-level directory, like a new component, add a rule for it too. Inside a component, `service` does the work and may use the database, but it doesn't import Connect. `api` turns Connect calls into calls to `service` and never reads the database itself. The authority's rules in `arch-go.yml` enforce this. `just arch` checks the rules, and `just ci` and the pre-commit hook run it. Note that arch-go reads packages like `go build` does, so it doesn't check imports in test files or in files behind a build tag.

Keep passwords, tokens and other secrets in a `secret.String` from `internal/secret`. It prints and logs as `****`, and only `Reveal()` returns the value. `just lint` fails if an exported field or parameter called something like `password` or `token` is a plain `string` instead.

Code outside tests doesn't call `crypto/rand`, `time.Now`, `time.Since` or `time.Until` itself. It takes a `random.Service` from [go-random](https://github.com/The127/go-random) or a `clock.Service` from [go-clock](https://github.com/The127/go-clock) as a dependency, and a test passes a mock, like `randomtest.NewMock()`. In production, use `random.NewCryptoRandom()`. tidegate uses no pseudo-random source outside tests. `just lint` fails on a call that breaks these rules.

The API is defined in `.proto` files in `proto/`. Every service, method, message, field and enum needs a comment, and `just proto::lint` fails without one. After you change a `.proto` file, run `just generate`. It writes the Go code to `gen/` and the [API reference](reference/api.md), built from the comments, to `docs/content/reference/`. Commit both. `just ci` fails if either doesn't match the `.proto` files.

A rule for a request field, like a maximum length, goes into the field's `(buf.validate.field)` option. [protovalidate](https://github.com/bufbuild/protovalidate) checks it before the handler runs, and a request that breaks it fails with `invalid_argument`. The API reference can't read these options, so also write the rule into the field's comment, as a last paragraph that starts with `Rules:`. The reference shows that paragraph in its own column.

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

`just test` and `just test-integration` run the tests with Go's [race detector](https://go.dev/doc/articles/race_detector).

The test recipes also measure test coverage and write it to `coverage/`, one file for each set of tests. To see which lines a set of tests runs, open its file:

```sh
go tool cover -html=coverage/authority-integration.out
```

CI uploads the files to [Codecov](https://codecov.io/gh/tidegate-dev/tidegate). Codecov shows the coverage on each pull request, and low coverage doesn't fail CI.

Coverage shows which lines the tests run, not whether they'd notice a mistake in them. To check that, run:

```sh
just dev-deps
just mutate
```

[gremlins](https://gremlins.dev) makes small changes to the lines your branch changed since it split off from `origin/main`, like `>` to `>=`. After each one, it runs the tests of that package. `LIVED` marks a change that no test caught, and `NOT COVERED` a line that no test of its package runs. Some changes can't be caught, because they don't change what the code does. `.gremlins.yaml` sets which kinds of changes gremlins makes.

The comparison includes uncommitted changes, but new files only once git tracks them. Run `git fetch` first, so `origin/main` is current. To compare with another branch or commit, name it, like `just mutate feat/login`.

On a pull request, CI shows the same list in the summary of the `ci` job. The list doesn't fail CI.

Fuzz tests are native Go fuzz tests, like `func FuzzPrepare(f *testing.F)`. `just test` runs only their seed inputs. To fuzz one locally, run `go test -fuzz=FuzzPrepare ./authority/internal/password`. In CI, [ClusterFuzzLite](https://google.github.io/clusterfuzzlite/) fuzzes for a minute on each pull request that changes Go code or the fuzz setup, and fails it when it finds a crash. The log shows the error, and the run keeps the input that caused it as an artifact named `crashes-` followed by the fuzz test's name. Every night it fuzzes for an hour and keeps the inputs it found for the next run. The fuzz tests share that time.

`.clusterfuzzlite/build.sh` finds the fuzz tests by their names, so a new one needs no extra setup. Put it in a test file of the package itself, not of a `_test` package, and without a `//go:build` line, because the fuzz build uses no build tags.

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
