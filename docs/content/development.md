# Development environment

The [README](https://github.com/tidegate-dev/tidegate#development) lists the tools you need and how to get them with Nix.

Once you have them, run `just` in the repository to see the recipes. `just ci` runs every check that CI runs.

To work on this site, run `just docs::serve` and open the address it prints. The page reloads when you change a file in `docs/content/`.
