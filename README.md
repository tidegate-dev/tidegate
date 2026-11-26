# phemera

[![CI](https://github.com/phemera-dev/phemera/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/phemera-dev/phemera/actions/workflows/ci.yaml?query=branch%3Amain)
[![Coverage](https://codecov.io/gh/phemera-dev/phemera/graph/badge.svg)](https://codecov.io/gh/phemera-dev/phemera)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/phemera-dev/phemera/badge)](https://scorecard.dev/viewer/?uri=github.com/phemera-dev/phemera)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/16081/badge)](https://www.bestpractices.dev/projects/16081)

An open source privileged access management tool.

The documentation is at [phemera-dev.github.io/phemera](https://phemera-dev.github.io/phemera/).

phemera is in early development and has no release yet. To try it, build it from source as described under [Development](#development).

## Platforms

The authority runs on Linux servers. The tools that people run on their own machines are built for Linux, macOS and Windows.

| Component | Shipped as | Platforms | Status |
|---|---|---|---|
| Authority | Binary and container image | Linux on amd64 and arm64 | In development |
| phemera CLI | Binary | Linux, macOS and Windows, each on amd64 and arm64 | In development |
| ssh-proxy | Binary and container image | Linux on amd64 and arm64 | Demo |

## Development

Development is supported on Linux and macOS. The Nix shell doesn't work on Intel Macs, because Nixpkgs no longer supports them.

Every setup also requires [Podman](https://podman.io/docs/installation) or [Docker](https://docs.docker.com/engine/install/), which the Nix shell doesn't provide. With Podman on macOS, `just kind` needs a rootful Podman machine.

If you use Nix and direnv, run `direnv allow` in the repository. It gives you a shell with every tool you need, in the versions the project uses. Without direnv, run `nix develop`.

The Nix shell also installs a pre-commit hook that runs `just lint`, `just test` and `just arch` and checks the staged changes with gitleaks. A commit with a lint error, a failing test, an import that `arch-go.yml` doesn't allow outside test files or a secret like a private key is refused.

If you don't use Nix, install these tools yourself:

- [actionlint](https://github.com/rhysd/actionlint) 1.7 or newer
- [Bats](https://bats-core.readthedocs.io/en/stable/installation.html) 1.14 or newer
- [buf](https://buf.build/docs/cli/installation/) 1.73 or newer
- [gitleaks](https://github.com/gitleaks/gitleaks) 8.30 or newer
- [Go](https://go.dev/dl/) 1.27.4 or newer
- [golangci-lint](https://golangci-lint.run/) 2.14 or newer
- [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) 1.9 or newer
- [grant](https://github.com/anchore/grant) 0.7 or newer
- [Helm](https://helm.sh/docs/intro/install/) 4.3 or newer
- [Hurl](https://hurl.dev/docs/installation.html) 8.0 or newer
- [jq](https://jqlang.org/download/) 1.8 or newer
- [just](https://just.systems/)
- [kind](https://kind.sigs.k8s.io/) 0.33 or newer
- [kubectl](https://kubernetes.io/docs/tasks/tools/) 1.37 or newer
- [OpenSSL](https://openssl-library.org/) 3 or newer
- [psql](https://www.postgresql.org/download/), the PostgreSQL client, 18 or newer
- [process-compose](https://f1bonacc1.github.io/process-compose/) 1.122 or newer
- [protoc-gen-go](https://pkg.go.dev/google.golang.org/protobuf/cmd/protoc-gen-go) 1.36 or newer
- [protoc-gen-connect-go](https://connectrpc.com/docs/go/getting-started/) 1.21 or a newer 1.x release
- [protoc-gen-doc](https://github.com/pseudomuto/protoc-gen-doc) 1.5 or newer
- [ShellCheck](https://github.com/koalaman/shellcheck) 0.11 or newer, which actionlint needs to check the shell scripts in the workflows
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html) 1.31 or newer
- [syft](https://github.com/anchore/syft) 1.55 or newer
- [Zensical](https://zensical.org/) 0.0.62 or newer, for the documentation site

Then run `just` to see the recipes, for example `just build` and `just ci`. Note that `just ci` needs the services from `just dev-deps`, like Postgres.

## Contributing

Report bugs and suggest features in [GitHub issues](https://github.com/phemera-dev/phemera/issues). Contributions are welcome. Before you open a pull request, read [CONTRIBUTING.md](CONTRIBUTING.md). It explains how to propose a change, how to title your pull request, how to sign off your commits and how you may use AI tools.

## Security

phemera controls access to infrastructure. If you find a vulnerability, don't open a public issue. Report it privately as described in [SECURITY.md](SECURITY.md).

## License

[Apache License 2.0](LICENSE)
