{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    devenv.url = "github:cachix/devenv";
  };

  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.devenv.flakeModule
      ];

      systems = [
        "x86_64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
        "aarch64-linux"
      ];

      perSystem =
        { pkgs, ... }:
        rec {
          devenv.shells = {
            default = {
              languages = {
                go = {
                  enable = true;
                  package = pkgs.go_1_24;
                };
              };

              packages = with pkgs; [
                gnumake

                golangci-lint
                gotestsum
                protobuf
                protoc-gen-go
                protoc-gen-go-grpc
                kind
              ];

              # https://github.com/cachix/devenv/issues/528#issuecomment-1556108767
              containers = pkgs.lib.mkForce { };
            };

            ci = devenv.shells.default;
          };
        };
    };
}
