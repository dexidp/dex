# Dex-client gRPC api

The main goal of this library is to interact with the gRPC api of dex, to write into Dex's database. Indead, Dex provides a gRPC service for programmatic modification of dex’s state. The API is intended to expose hooks for management applications and is not expected to be used by most installations.

This API is able to create, read, update and delete data into *user_idp*, *user*, *acl_token* and *client_token* tables.

The last but not least, this library interact with first authentification funtcions/way in Dex. Indead, we are modifying Dex to be able to give a first auth way, if you will log for the first time you need to provide an Acl Token. (see (HowToUse Doc[HowToUse.md])

## Configuration

Into the config file we need to specialise some credentials path to trust the application's server. All fil can be produced by the service of Dex

### Generate credantial

At the root of Dex project run this:

```bash
export SAN=IP.1:{Server_IP}
./examples/grpc-client/cert-gen
```

It will create some fils like:
- ca.crt
- ca.key
- client.ca
- client.key
- server.ca
- server.key

To be sure that all files are good run:

```bash
openssl verify -CAfile ca.crt server.crt
openssl verify -CAfile ca.crt client.crt
```

Then copy/past then into a directory, for example at : {HOME}.dex/certs/{server_IP}/


### Change the Dex configuration file yaml

To expose the gRPC service, the gRPC option must be enabled via the dex config file as shown below.

```yaml
grpc:
  addr: {server_IP}
  tlsCert: {HOME}.dex/certs/{server_IP}/server.crt
  tlsKey: {HOME}.dex/certs/{server_IP}/server.key
  tlsCLientCA: {HOME}.dex/certs/{server_IP}/client.crt
```

### Run the dex server

```bash
./bin/dex server examples/grpc-client/config.yaml
```

### Run the dex client

```bash
./dex-client --ca-crt={HOME}/.dex/certs/{server_IP}/ca.crt --client-crt={HOME}/.dex/certs/{server_IP}/client.crt --client-key={HOME}/.dex/certs/{server_IP}/client.key
```

## Methods 

Here are the list of methods found by gRPC api dex:

- CreateIdpUser() 
- UpdateIdpUser()
- GetIdpUser()
- DeleteIdpUser()

Same function for the other tables

This project will use this method to create a library more usable. Here is a example of what we can do :

- **NewGrpcApiDex()**

```Go
	// Catch client certifications
	caCrt := flag.String("ca-crt", "", "CA certificate")
	clientCrt := flag.String("client-crt", "", "Client certificate")
	clientKey := flag.String("client-key", "", "Client key")
	flag.Parse()
	if *clientCrt == "" || *caCrt == "" || *clientKey == "" {
		log.Fatal("Please provide CA & client certificates and client key. Usage: ./bin/dex-client --ca-crt=<path ca.crt> --client-crt=<path client.crt> --client-key=<path client key>")
	}

	// Allocate a grpcApi
	grpcApi, err := grpc.NewGrpcApiDex("10.153.191.9:5557", *caCrt, *clientCrt, *clientKey)
	if err != nil {
		log.Fatalf("Failed to allocate grpcApi struct: %v\n", err)
	}
```