# tidegate

[![CI](https://github.com/tidegate-dev/tidegate/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/tidegate-dev/tidegate/actions/workflows/ci.yaml?query=branch%3Amain)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/tidegate-dev/tidegate/badge)](https://scorecard.dev/viewer/?uri=github.com/tidegate-dev/tidegate)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/16081/badge)](https://www.bestpractices.dev/projects/16081)

An open source privileged access management tool.

The documentation is at [tidegate-dev.github.io/tidegate](https://tidegate-dev.github.io/tidegate/).

## Platforms

The authority runs on Linux servers. The tools that people run on their own machines are built for Linux, macOS and Windows.

| Component | Shipped as | Platforms | Status |
|---|---|---|---|
| Authority | Binary and container image | Linux on amd64 and arm64 | In development |
| `tidegate` CLI | Binary | Linux, macOS and Windows, each on amd64 and arm64 | In development |

## Development

Development is supported on Linux and macOS. The Nix shell doesn't work on Intel Macs, because Nixpkgs no longer supports them.

Every setup also requires [Podman](https://podman.io/docs/installation) or [Docker](https://docs.docker.com/engine/install/), which the Nix shell doesn't provide. With Podman on macOS, `just kind` needs a rootful Podman machine.

If you use Nix and direnv, run `direnv allow` in the repository. It gives you a shell with every tool you need, in the versions the project uses. Without direnv, run `nix develop`.

The Nix shell also installs a pre-commit hook that runs `just lint`, `just test` and `just arch` and checks the staged changes with gitleaks. A commit with a lint error, a failing test, an import that `arch-go.yml` doesn't allow outside test files or a secret like a private key is refused.

If you don't use Nix, install these tools yourself:

- [buf](https://buf.build/docs/cli/installation/) 1.73 or newer
- [gitleaks](https://github.com/gitleaks/gitleaks) 8.30 or newer
- [Go](https://go.dev/dl/) 1.27.3 or newer
- [golangci-lint](https://golangci-lint.run/) 2.14 or newer
- [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) 1.9 or newer
- [Helm](https://helm.sh/docs/intro/install/) 4.3 or newer
- [just](https://just.systems/)
- [kind](https://kind.sigs.k8s.io/) 0.33 or newer
- [kubectl](https://kubernetes.io/docs/tasks/tools/) 1.37 or newer
- [OpenSSL](https://openssl-library.org/) 3 or newer
- [process-compose](https://f1bonacc1.github.io/process-compose/) 1.122 or newer
- [protoc-gen-go](https://pkg.go.dev/google.golang.org/protobuf/cmd/protoc-gen-go) 1.36 or newer
- [protoc-gen-connect-go](https://connectrpc.com/docs/go/getting-started/) 1.21 or a newer 1.x release
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html) 1.31 or newer
- [Zensical](https://zensical.org/) 0.0.62 or newer, for the documentation site

Then run `just` to see the recipes, for example `just build` and `just ci`. Note that `just ci` needs the services from `just dev-deps`, like Postgres.

## Contributing

Contributions are welcome. Before you open a pull request, read [CONTRIBUTING.md](CONTRIBUTING.md). It explains how to propose a change, how to title your pull request, how to sign off your commits and how you may use AI tools.

## License

[Apache License 2.0](LICENSE)
