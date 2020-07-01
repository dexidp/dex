# The Dex API

Dex provides a [gRPC](http://www.grpc.io/) service for programmatic modification of dex's state.
The API is intended to expose hooks for management applications and is not expected to be used by most installations.

This document is an overview of how to interact with the API.


## Configuration

Admins that wish to expose the gRPC service must add the following entry to the dex config file. This option is off by default.

```yaml
grpc:
  # Cannot be the same address as an HTTP(S) service.
  addr: 127.0.0.1:5557

  # Server certs. If TLS credentials aren't provided dex will run in plaintext (HTTP) mode.
  tlsCert: /etc/dex/grpc.crt
  tlsKey: /etc/dex/grpc.key

  # Client auth CA.
  tlsClientCA: /etc/dex/client.crt

  # enable reflection
  reflection: true
```


## Clients

gRPC is a suite of tools for generating client and server bindings from a common declarative language.
The canonical schema for Dex's API can be found in the source tree at [`api/v2/api.proto`](../api/v2/api.proto).
Go bindings are generated and maintained in the same directory for both public and internal use.


### Go

A Go project can import the API module directly, without having to import the entire project:

```bash
go get github.com/dexidp/dex/api/v2
```

The client then can be used as follows:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/dexidp/dex/api/v2"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)

func newDexClient(hostAndPort, caPath string) (api.DexClient, error) {
    creds, err := credentials.NewClientTLSFromFile(caPath, "")
    if err != nil {
        return nil, fmt.Errorf("load dex cert: %v", err)
    }

    conn, err := grpc.Dial(hostAndPort, grpc.WithTransportCredentials(creds))
    if err != nil {
        return nil, fmt.Errorf("dial: %v", err)
    }
    return api.NewDexClient(conn), nil
}

func main() {
    client, err := newDexClient("127.0.0.1:5557", "/etc/dex/grpc.crt")
    if err != nil {
        log.Fatalf("failed creating dex client: %v ", err)
    }

    req := &api.CreateClientReq{
        Client: &api.Client{
            Id:           "example-app",
            Name:         "Example App",
            Secret:       "ZXhhbXBsZS1hcHAtc2VjcmV0",
            RedirectUris: []string{"http://127.0.0.1:5555/callback"},
        },
    }

    if _, err := client.CreateClient(context.TODO(), req); err != nil {
        log.Fatalf("failed creating oauth2 client: %v", err)
    }
}
```

A clear working example of the Dex gRPC client for Go can be found [here](../examples/grpc-client/README.md).


### Other languages

To generate a client for your own project install [`protoc`](https://github.com/google/protobuf/releases),
install a protobuf generator for your project's language, and download the `api.proto` file.

Here is an example:

```bash
# Download api.proto for a given version.
$ DEX_VERSION=v2.24.0
$ wget https://raw.githubusercontent.com/dexidp/dex/${DEX_VERSION}/api/v2/api.proto

# Generate the client bindings.
$ protoc [YOUR LANG PARAMS] api.proto
```

Client programs can then be written using the generated code.


## Authentication and access control

The Dex API does not provide any authentication or authorization beyond TLS client auth.

Projects that wish to add access controls on top of the existing API should build apps which perform such checks.
For example to provide a "Change password" screen, a client app could use Dex's OpenID Connect flow to authenticate an end user,
then call Dex's API to update that user's password.


## dexctl?

Dex does not ship with a command line tool for interacting with the API.
Command line tools are useful but hard to version, easy to design poorly,
and expose another interface which can never be changed in the name of compatibility.

While the Dex team would be open to re-implementing `dexctl` for v2 a majority of the work is writing a design document,
not the actual programming effort.


## Why not REST or gRPC Gateway?

Between v1 and v2, Dex switched from REST to gRPC. This largely stemmed from problems generating documentation,
client bindings, and server frameworks that adequately expressed REST semantics.
While [Google APIs](https://github.com/google/apis-client-generator), [Open API/Swagger](https://openapis.org/),
and [gRPC Gateway](https://github.com/grpc-ecosystem/grpc-gateway) were evaluated,
they often became clunky when trying to use specific HTTP error codes or complex request bodies.
As a result, v2's API is entirely gRPC.

Many arguments _against_ gRPC cite short term convenience rather than production use cases.
Though this is a recognized shortcoming, Dex already implements many features for developer convenience.
For instance, users who wish to manually edit clients during testing can use the `staticClients` config field instead of the API.
