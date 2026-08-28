{ inputs, ... }:
let
  version = "0.1.0";
  rev = inputs.self.rev or inputs.self.dirtyRev or "unknown";
in
{
  perSystem =
    {
      pkgs,
      lib,
      self',
      ...
    }:
    {
      packages.default = pkgs.buildGo127Module {
        meta = {
          description = "A custom CI -> AWS auth mechanism powering my personal software catalog";
          homepage = "https://github.com/FollowTheProcess/gatekeeper";
          license = lib.licenses.mit;
          platforms = lib.platforms.unix;
          mainProgram = "gatekeeper";
        };

        pname = "gatekeeper";
        inherit version;
        src = lib.sources.cleanSource inputs.self;
        vendorHash = "sha256-Ns9zobFfX1rl8gpOCFKd1jgTCdY6OkWvErNoTNJABuQ=";
        ldflags = [
          "-s"
          "-w"
          "-X go.followtheprocess.codes/gatekeeper/internal/cmd.version=${version}"
          "-X go.followtheprocess.codes/gatekeeper/internal/cmd.commit=${rev}"
          "-X go.followtheprocess.codes/gatekeeper/internal/cmd.date=${inputs.self.lastModifiedDate}"
        ];

        env = {
          CGO_ENABLED = 0;
        };

        checkPhase = ''
          runHook preCheck
          CGO_ENABLED=1 go test -race ./...
          runHook postCheck
        '';
      };

      checks.gatekeeper = self'.packages.default;
    };
}
