# Running a Dex gRPC client

Using gRPC, a client application can directly call methods on a server application as if it was a local object. The schema for Dex's gRPC API calls is defined in [`api/api.proto`][api-proto]. [`client.go`][client] is an example client program that makes a bunch of API calls to the dex server. For further details on the Dex API refer [`Documentation/api.md`][api-docs].

## Generating Credentials

Before running the client or the server, TLS credentials have to be setup for secure communication. Run the `cred-gen` script to create TLS credentials for running this example. This script generates a `ca.crt`, `server.crt`, `server.key`, `client.crt`, and `client.key`.

```
# Used to set certificate subject alt names.
export SAN=IP.1:127.0.0.1

# Run the script
./examples/grpc-client/cert-gen
```
To verify that the server and client certificates were signed by the CA, run the following commands:

```
openssl verify -CAfile ca.crt server.crt
openssl verify -CAfile ca.crt client.crt
```

## Running the Dex server

To expose the gRPC service, the gRPC option must be enabled via the dex config file as shown below.

```yaml
# Enables the gRPC API.
grpc:
  addr: 127.0.0.1:5557
  tlsCert: server.crt
  tlsKey: server.key

```
Start an instance of the dex server with an in-memory data store:

```
./bin/dex serve examples/grpc-client/config.yaml
```

## Running the Dex client

Finally run the Dex client providing the CA certificate, client certificate and client key as arguments.

```
./bin/grpc-client -ca-crt=ca.crt -client-crt=client.crt -client-key=client.key
```
Running the gRPC client will cause the following API calls to be made to the server
1. CreatePassword
2. ListPasswords
3. VerifyPassword
4. DeletePassword

## Cleaning up

Run the following command to destroy all the credentials files that were created by the `cert-gen` script:

```
./examples/grpc-client/cert-destroy
```
[api-proto]: ../../api/api.proto
[client]: client.go
[api-docs]: ../../Documentation/api.md
