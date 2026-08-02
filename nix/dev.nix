_: {
  perSystem = { pkgs, ... }: {
    devShells.default = pkgs.mkShell {
      packages = with pkgs; [
        aws-sam-cli
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
}
