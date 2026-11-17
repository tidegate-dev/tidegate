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
        pkgs.actionlint
        pkgs.buf
        pkgs.go_1_27
        pkgs.golangci-lint
        pkgs.just
      ];

      # nixpkgs doesn't package grant, so cache.nixos.org has no build of it to download.
      grant =
        pkgs:
        let
          releases = {
            x86_64-linux = {
              platform = "linux_amd64";
              hash = "sha256-4CEj6pKGYR0Bul/y1xujQieZFAKsWqZo3B0fQFc4Jfc=";
            };
            aarch64-linux = {
              platform = "linux_arm64";
              hash = "sha256-zAhggkGNN1y3gU5RChj+1f5ZIWQnMJlZXGZyvKiVff0=";
            };
            aarch64-darwin = {
              platform = "darwin_arm64";
              hash = "sha256-q/f8UhzwcrnYtBPiR8iFMExvsnUNJ0awIEu0wPyXMuc=";
            };
          };
          release = releases.${pkgs.stdenv.hostPlatform.system};
        in
        pkgs.stdenvNoCC.mkDerivation (finalAttrs: {
          pname = "grant";
          version = "0.7.1";
          src = pkgs.fetchurl {
            url = "https://github.com/anchore/grant/releases/download/v${finalAttrs.version}/grant_${finalAttrs.version}_${release.platform}.tar.gz";
            inherit (release) hash;
          };
          sourceRoot = ".";
          installPhase = ''
            install -D -m 755 grant $out/bin/grant
          '';
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
