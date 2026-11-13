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

      # nixpkgs doesn't package grant.
      grant =
        pkgs:
        pkgs.buildGoModule (finalAttrs: {
          pname = "grant";
          version = "0.7.1";
          src = pkgs.fetchFromGitHub {
            owner = "anchore";
            repo = "grant";
            tag = "v${finalAttrs.version}";
            hash = "sha256-Zznac28Gb7b3+uMzUbzPASs27SpYv8i75QzydTKtjhk=";
          };
          vendorHash = "sha256-2d+G5z+DS9BDqO8WhogTD019qJZVNcj6xYLctFIBC00=";
          subPackages = [ "cmd/grant" ];
          ldflags = [
            "-s"
            "-w"
            "-X main.version=${finalAttrs.version}"
          ];
        });

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
            arch = {
              enable = true;
              name = "just arch";
              entry = toString (
                pkgs.writeShellScript "just-arch" ''
                  export PATH=${pkgs.lib.makeBinPath (tools pkgs)}:$PATH
                  just arch
                ''
              );
              pass_filenames = false;
            };
            secrets = {
              enable = true;
              name = "gitleaks";
              entry = "${pkgs.gitleaks}/bin/gitleaks git --pre-commit --staged --redact --no-banner";
              pass_filenames = false;
            };
          };
        };
    in
    {
      devShells = forEachSystem (pkgs: {
        default = pkgs.mkShell {
          packages = tools pkgs ++ [
            pkgs.bats
            pkgs.gitleaks
            pkgs.govulncheck
            (grant pkgs)
            pkgs.hurl
            pkgs.jq
            pkgs.kind
            pkgs.kubectl
            pkgs.kubernetes-helm
            pkgs.openssl
            # pkgs.postgresql would also add its headers, which pull in LLVM, thats why we add `.out`.
            pkgs.postgresql.out
            pkgs.process-compose
            pkgs.protoc-gen-connect-go
            pkgs.protoc-gen-doc
            pkgs.protoc-gen-go
            pkgs.sqlc
            pkgs.syft
            pkgs.zensical
          ];
          shellHook = (hooks pkgs).shellHook;
        };
      });

      formatter = forEachSystem (pkgs: pkgs.nixfmt);
    };
}
