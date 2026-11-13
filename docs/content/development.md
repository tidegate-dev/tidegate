# Development

This page is for people who work on tidegate. It explains how to run tidegate from your working copy, which rules the code follows and how to test a change.

## Set up

The [README](https://github.com/tidegate-dev/tidegate#development) lists the tools you need and how to get them with Nix. Once you have them, run `just` in the repository to see the recipes.

Before you open a pull request, run `just ci`. It runs every check that CI runs, except fuzzing and mutation testing, which are described under [Tests](#tests). Its tests need Postgres, so start it first with `just dev-deps`.

To work on this site, run `just docs::serve` and open the address it prints. The page reloads when you change a file in `docs/content/`.

## Run tidegate locally

```sh
just dev
```

This starts tidegate's components and the services they need, like Postgres, with [process-compose](https://f1bonacc1.github.io/process-compose/). The components run from your working copy. After you change one, restart it in process-compose, and it builds again.

To debug a component, stop it in process-compose and start it in your editor's debugger. To start only the services and run the components yourself, use:

```sh
just dev-deps
```

The services keep their data between runs. To start with empty data, stop `just dev` or `just dev-deps` and run `just dev-reset`. `dev/tidegate.yaml` defines the components, and `dev/deps.yaml` defines the services with their ports and logins.

One of the services is an SSH host to try tidegate against. Connect to it with the password `tidegate`:

```sh
ssh -p 2222 tidegate@127.0.0.1
```

Its host key is committed in `dev/ssh-host/`, so the host has the same fingerprint for everyone.

## Run tidegate in Kubernetes

To run tidegate in a local Kubernetes cluster with [kind](https://kind.sigs.k8s.io/), use:

```sh
just kind
```

This builds the components' container images, creates the cluster if it doesn't exist yet, installs cert-manager with Helm and deploys everything in `dev/kind/`. After a change, run it again. Note that kind also switches your kubectl context to the new cluster. To delete the cluster, run `just kind-down`.

## Trust the local CA

Both `just dev` and `just kind` serve the authority over HTTPS on port 8080, with a certificate from a local CA. They create the CA in `dev/tls/` the first time, and in `just kind`, cert-manager issues the authority's certificate from it. To trust the CA, point `SSL_CERT_FILE` to it:

```sh
SSL_CERT_FILE=dev/tls/ca.pem TIDEGATE_SERVER=https://127.0.0.1:8080 tidegate ping
```

Other tools take the CA file as a flag, like `buf curl --cacert dev/tls/ca.pem`. Your system's trust store stays unchanged.

## Code rules

`just lint`, `just arch` and `just proto::lint` check most of the rules below, so CI fails when a change breaks one.

### Imports

`arch-go.yml` says which packages may import which. Shared code in `internal/` imports nothing from a component or from `gen/`. A component imports only its own packages, `internal/` and the generated code in `gen/`. Every package needs a rule in `arch-go.yml`, so when you add a top-level directory, like a new component, add a rule for it too.

Inside a component, `service` does the work and may use the database, but doesn't import Connect. `api` turns Connect calls into calls to `service` and never reads the database itself. The authority's rules in `arch-go.yml` enforce this split.

`just arch` checks the rules, and `just ci` and the pre-commit hook run it. Note that arch-go reads packages like `go build` does, so it doesn't check imports in test files or in files behind a build tag.

### Secrets

Keep passwords, tokens and other secrets in a `secret.String` from `internal/secret`. It prints and logs as `****`, and only `Reveal()` returns the value. `just lint` fails if an exported field or parameter with a name like `password` or `token` is a plain `string`.

### Time and randomness

Outside tests, don't call `crypto/rand`, `time.Now`, `time.Since` or `time.Until` directly. Take a `random.Service` from [go-random](https://github.com/The127/go-random) or a `clock.Service` from [go-clock](https://github.com/The127/go-clock) as a dependency instead, so a test can pass a mock like `randomtest.NewMock()`. In production code, use `random.NewCryptoRandom()`. tidegate uses no pseudo-random source outside tests. `just lint` fails on a call that breaks these rules.

### API

The API is defined in `.proto` files in `proto/`. Every service, method, message, field and enum needs a comment, and `just proto::lint` fails without one.

After you change a `.proto` file, run `just generate`. It writes the Go code to `gen/` and the [API reference](reference/api.md), built from the comments, to `docs/content/reference/`. Commit both, because `just ci` fails if either doesn't match the `.proto` files.

Put a rule for a request field, like a maximum length, into the field's `(buf.validate.field)` option. [protovalidate](https://github.com/bufbuild/protovalidate) checks it before the handler runs, and a request that breaks it fails with `invalid_argument`. The API reference can't read these options, so also name the rule in the field's comment, in a last paragraph that starts with `Rules:`:

```proto
// Password is the user's local password.
//
// Rules: Not empty, at most 1024 bytes.
string password = 2 [(buf.validate.field).string = {
  min_len: 1
  max_bytes: 1024
}];
```

The reference shows that paragraph in its own column.

### Database

The authority's SQL queries are in `authority/internal/db/queries/`, and sqlc generates Go code for them in `authority/internal/db/sqlc/`. After you change a query, run `just generate` and commit the result.

To change the database schema, add a migration to `authority/internal/db/migrations/`, numbered after the last one. A migration only goes forward and only adds to the schema. To remove a table or a column, wait a release after the code stops using it, so an old and a new authority can share the database during an update. `just dev` and `just kind` migrate when the authority starts.

## Tests

tidegate has unit tests, integration tests, API tests and end-to-end tests, and CI runs all four. CI also measures their coverage, lists the changes that no test catches and fuzzes the code.

### Unit and integration tests

Unit tests use the standard `testing` package with testify. Integration tests are [Ginkgo](https://onsi.github.io/ginkgo/) specs with [Gomega](https://onsi.github.io/gomega/) assertions. Their files start with `//go:build integration`, so `just test` and `go test ./...` leave them out. To create a database for a spec, call `dbtest.NewDatabase` from `authority/internal/dbtest`.

The integration tests run against Postgres. Start the services first, then run the tests:

```sh
just dev-deps
just test-integration
```

Each test creates its own database and drops it at the end. If you stop a run before it finishes, its databases stay behind, and `just dev-reset` deletes them. To use another Postgres server, set `TIDEGATE_TEST_DATABASE_URL` to its URL. Its user needs the `CREATEDB` privilege.

`just test` and `just test-integration` run the tests with Go's [race detector](https://go.dev/doc/articles/race_detector).

### API tests

API tests send requests to a running component with [Hurl](https://hurl.dev) and check what comes back, like the status code and the error. They check what a client sees, and the Go tests check the logic behind it. Each component keeps its API tests in its own `api-tests/` folder, like `authority/api-tests/`. Start the services first, then run the tests:

```sh
just dev-deps
just test-api
```

This creates a database of its own on the test Postgres server, starts the authority against it on other ports than `just dev` and runs the tests. When the tests end, it drops the database again, so the tests never see the data from `just dev` or from an earlier run.

### End-to-end tests

End-to-end tests use tidegate the way a user does. They run its binaries with [Bats](https://bats-core.readthedocs.io/) and check what they print. The tests in `e2e/cli` run the `tidegate` CLI against a running authority. The tests in `e2e/authority` run the commands of the `authority` binary, like `authority create-initial-admin`. Start the services first, then run the tests:

```sh
just dev-deps
just test-e2e
```

Like `just test-api`, this starts the authority on a database of its own and creates the initial admin. A test finds the authority in `TIDEGATE_TEST_AUTHORITY` and the admin's user name and password in `TIDEGATE_TEST_USER` and `TIDEGATE_TEST_PASSWORD`. If a test needs an empty database, it creates one in `setup` and drops it in `teardown`. `e2e/authority/create-initial-admin.bats` shows how.

### Coverage

`just test`, `just test-integration`, `just test-api` and `just test-e2e` measure coverage and write it to `coverage/`, one file for each set of tests. To see which lines a set of tests runs, open its file:

```sh
go tool cover -html=coverage/authority-integration.out
```

CI uploads the files to [Codecov](https://codecov.io/gh/tidegate-dev/tidegate). Codecov shows the coverage on each pull request, and low coverage doesn't fail CI.

### Mutation testing

Coverage shows which lines the tests run, not whether they'd notice a mistake in them. To check that, run:

```sh
just dev-deps
just mutate
```

[gremlins](https://gremlins.dev) makes small changes to the lines your branch changed since it split off from `origin/main`, like `>` to `>=`. After each change, it runs the tests of that package. `LIVED` marks a change that no test caught, and `NOT COVERED` a line that no test of its package runs. Some changes can't be caught, because they don't change what the code does. `.gremlins.yaml` sets which kinds of changes gremlins makes.

The comparison includes uncommitted changes, but new files only once git tracks them. Run `git fetch` first, so `origin/main` is current. To compare with another branch or commit, name it, like `just mutate feat/login`.

On a pull request, CI shows the same list in the summary of the `ci` job. The list doesn't fail CI.

### Fuzzing

Fuzz tests are native Go fuzz tests, like `func FuzzPrepare(f *testing.F)`. `just test` runs only their seed inputs. To fuzz one locally, run:

```sh
go test -fuzz=FuzzPrepare ./authority/internal/password
```

In CI, [ClusterFuzzLite](https://google.github.io/clusterfuzzlite/) fuzzes for a minute on each pull request that changes Go code or the fuzz setup, and fails it when it finds a crash. The log shows the error, and the run keeps the input that caused it as an artifact named `crashes-` followed by the fuzz test's name. Every night it fuzzes for an hour, shared by all fuzz tests, and keeps the inputs it found for the next run.

`.clusterfuzzlite/build.sh` finds fuzz tests by their names, so a new one needs no extra setup. Put it in a test file of the package itself, not of a `_test` package, and without a `//go:build` line, because the fuzz build uses no build tags.
