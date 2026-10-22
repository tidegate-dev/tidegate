{
  description = "tidegate development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    git-hooks = {
      url = "github:cachix/git-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    { nixpkgs, git-hooks, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      forEachSystem = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      # When you change this list, update the tool list in README.md too.
      tools = pkgs: [
        pkgs.buf
        pkgs.go_1_27
        pkgs.golangci-lint
        pkgs.just
      ];

      hooks =
        pkgs:
        git-hooks.lib.${pkgs.stdenv.hostPlatform.system}.run {
          src = ./.;
          hooks = {
            just-lint-test = {
              enable = true;
              name = "just lint test";
              # Commits from an editor or a GUI don't run in the dev shell, so
              # the hook brings its own tools.
              entry = toString (
                pkgs.writeShellScript "just-lint-test" ''
                  export PATH=${pkgs.lib.makeBinPath (tools pkgs)}:$PATH
                  just lint test
                ''
              );
              pass_filenames = false;
            };
          };
        };
    in
    {
      devShells = forEachSystem (pkgs: {
        default = pkgs.mkShell {
          packages = tools pkgs ++ [
            pkgs.govulncheck
            pkgs.process-compose
            pkgs.protoc-gen-connect-go
            pkgs.protoc-gen-go
            pkgs.zensical
          ];
          shellHook = (hooks pkgs).shellHook;
        };
      });

      formatter = forEachSystem (pkgs: pkgs.nixfmt);
    };
}
