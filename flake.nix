{
  description = "A custom CI -> AWS auth mechanism powering my personal software catalog";

  inputs = {
    nixpkgs.url = "https://channels.nixos.org/nixpkgs-unstable/nixexprs.tar.xz";
    flake-parts.url = "github:hercules-ci/flake-parts";
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [ inputs.treefmt-nix.flakeModule ];

      systems = [
        "aarch64-linux"
        "x86_64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];

      perSystem =
        {
          self',
          pkgs,
          lib,
          ...
        }:
        let
          version = "0.1.0";
          rev = inputs.self.rev or inputs.self.dirtyRev or "unknown";
        in
        {
          treefmt.config = {
            programs = {
              deadnix.enable = true;
              gofmt.enable = true;
              nixfmt.enable = true;
              shellcheck.enable = true;
              shfmt.enable = true;
              statix.enable = true;
              yamlfmt = {
                enable = true;
                settings.formatter = {
                  type = "basic";
                  eof_newline = true;
                  indent = 2;
                  pad_line_comments = 2;
                  retain_line_breaks_single = true;
                  scan_folded_as_literal = true;
                  trim_trailing_whitespace = true;
                };
              };
            };
          };

          packages.default = pkgs.buildGoModule {
            meta = {
              description = "A custom CI -> AWS auth mechanism powering my personal software catalog";
              homepage = "https://github.com/FollowTheProcess/gatekeeper";
              license = lib.licenses.mit;
              platforms = lib.platforms.unix;
              mainProgram = "gatekeeper";
            };

            pname = "gatekeeper";
            inherit version;
            src = ./.;
            vendorHash = "sha256-FZtnUmBT6qvljFLLDMQiMQWojTSjiwZTB6rh6M09BaE=";
            ldflags = [
              "-s"
              "-w"
              "-X go.followtheprocess.codes/gatekeeper/internal/cmd.version=${version}"
              "-X go.followtheprocess.codes/gatekeeper/internal/cmd.commit=${rev}"
              "-X go.followtheprocess.codes/gatekeeper/internal/cmd.date=${inputs.self.lastModifiedDate}"
            ];
            subPackages = [ "cmd/gatekeeper" ];

            env = {
              CGO_ENABLED = 0;
              GOEXPERIMENT = "jsonv2";
            };

            checkPhase = ''
              runHook preCheck
              CGO_ENABLED=1 go test -race ./...
              runHook postCheck
            '';
          };

          checks.gatekeeper = self'.packages.default;

          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              aws-sam-cli
              # checkov
              go
              golangci-lint
              gopls
              goreleaser
              mise
              nix-update
              trivy
              typos
            ];

            GOEXPERIMENT = "jsonv2";

            shellHook = ''
              echo "👋 Welcome to the gatekeeper devShell!"
            '';
          };
        };
    };
}
