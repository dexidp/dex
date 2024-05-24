{
  description = "OpenID Connect (OIDC) identity and OAuth 2.0 provider with pluggable connectors";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        buildDeps = with pkgs; [ git go_1_21 gnumake ];
        devDeps = with pkgs;
          buildDeps ++ [
            golangci-lint
            gotestsum
            protobuf
            protoc-gen-go
            protoc-gen-go-grpc
            kind
          ];
      in
      { devShell = pkgs.mkShell { buildInputs = devDeps; }; }
    );
}
