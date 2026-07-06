{
  pkgs,
  ...
}:

{
  dotenv.enable = true;

  packages = with pkgs; [
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
    };
  };
}
