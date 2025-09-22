{ pkgs, inputs, ... }:

{
  overlays = [
    (final: prev: {
      dagger = inputs.dagger.packages.${final.system}.dagger;
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
