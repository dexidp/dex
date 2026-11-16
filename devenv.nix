{
  pkgs,
  ...
}:

{
  dotenv.enable = true;

  packages = with pkgs; [
    golangci-lint

    gnumake

    gotestsum
    protobuf
    protoc-gen-go
    protoc-gen-go-grpc
    kind
  ];

  languages = {
    go = {
      enable = true;
      package = pkgs.go_1_27;
    };
  };
}
