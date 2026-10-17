# tidegate

An open source privileged access management tool.

The documentation is at [tidegate-dev.github.io/tidegate](https://tidegate-dev.github.io/tidegate/).

## Development

Development is supported on Linux and macOS. The Nix shell doesn't work on Intel Macs, because Nixpkgs no longer supports them.

If you use Nix and direnv, run `direnv allow` in the repository. It gives you a shell with every tool you need, in the versions the project uses. Without direnv, run `nix develop`.

The Nix shell also installs a pre-commit hook that runs `just lint` and `just test`, so a commit with a lint error or a failing test is refused.

If you don't use Nix, install these tools yourself:

- [Go](https://go.dev/dl/) 1.27.2 or newer
- [golangci-lint](https://golangci-lint.run/) 2.14 or newer
- [just](https://just.systems/)
- [Zensical](https://zensical.org/) 0.0.62 or newer, for the documentation site

Then run `just` to see the recipes, for example `just build` and `just ci`.

## Contributing

Contributions are welcome. Before you open a pull request, read [CONTRIBUTING.md](CONTRIBUTING.md). It explains how to propose a change, how to title your pull request, how to sign off your commits and how you may use AI tools.

## License

[Apache License 2.0](LICENSE)
