_: {
  perSystem = { pkgs, ... }: {
    devShells.default = pkgs.mkShell {
      packages = with pkgs; [
        aws-sam-cli
        go_1_27
        golangci-lint
        gopls
        goreleaser
        just
        nix-update
        trivy
        typos
      ];

      shellHook = ''
        echo "👋 Welcome to the gatekeeper devShell!"
      '';
    };
  };
}
