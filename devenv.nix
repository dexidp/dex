{ pkgs, inputs, ... }:

{
  overlays = [
    (final: prev: {
      go_1_25 = inputs.unstable.legacyPackages.${final.system}.go_1_25;
    })
  ];

  languages = {
    go = {
      enable = true;
      package = pkgs.go_1_25;
    };
  };

  packages = with pkgs; [
    just
    gnumake
    golangci-lint
    gotestsum
    protobuf
    protoc-gen-go
    protoc-gen-go-grpc
    kind
    dagger
  ];
}
