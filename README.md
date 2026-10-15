# tidegate

An open source privileged access management tool.

## Development

Development is supported on Linux and macOS. The Nix shell doesn't work on Intel Macs, because Nixpkgs no longer supports them.

If you use Nix and direnv, run `direnv allow` in the repository. It gives you a shell with every tool you need, in the versions the project uses. Without direnv, run `nix develop`.

If you don't use Nix, install these tools yourself:

- [Go](https://go.dev/dl/) 1.27.2 or newer
- [just](https://just.systems/)

Then run `just` to see the recipes, for example `just build` and `just ci`.

## Contributing

Contributions are welcome. Before you open a pull request, read [CONTRIBUTING.md](CONTRIBUTING.md). It explains how to propose a change, how to title your pull request, how to sign off your commits and how you may use AI tools.

## License

[Apache License 2.0](LICENSE)
