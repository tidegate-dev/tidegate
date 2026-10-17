{
  description = "tidegate development environment";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { nixpkgs, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      forEachSystem = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      devShells = forEachSystem (pkgs: {
        default = pkgs.mkShell {
          # When you change this list, update the tool list in README.md too.
          packages = [
            pkgs.go_1_27
            pkgs.golangci-lint
            pkgs.just
          ];
        };
      });

      formatter = forEachSystem (pkgs: pkgs.nixfmt);
    };
}
