// Package apiserver implements the gRPC management API (api.DexServer): the CRUD
// and administrative calls for clients, passwords, connectors, refresh tokens,
// auth sessions, user identities and MFA devices. Each domain lives in its own
// file. It depends only on storage, the connector cache and a discovery-document
// builder, not on the whole Server.
package apiserver
