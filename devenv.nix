{
  pkgs,
  ...
}:

let
  # go.mod requires Go 1.27, newer than nixpkgs' default `go` (1.26).
  go = pkgs.go_1_27;
in
{
  dotenv.enable = true;

  # `languages.go` is deliberately not used: it rebuilds gopls, delve, gotools,
  # go-tools, gomodifytags, impl, gotests and iferr against the selected Go
  # package. When that package is not nixpkgs' default `go`, none of those
  # builds are in the binary cache and they all compile on first init.
  # Every package below is fetched prebuilt from cache.nixos.org.
  packages = with pkgs; [
    go
    gopls # built with buildGoLatestModule (Go 1.27) in nixpkgs
    golangci-lint # built with buildGo127Module in nixpkgs

    gnumake

    gotestsum
    protobuf
    protoc-gen-go
    protoc-gen-go-grpc
    kind
  ];

  env = {
    GOROOT = "${go}/share/go";
    GOTOOLCHAIN = "local";
  };
}
