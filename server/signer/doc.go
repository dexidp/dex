// Package signer signs tokens and exposes the corresponding validation keys.
// It supports pluggable key backends (in-memory local rotation and Vault) and a
// KeySet that verifies JWT signatures against the current validation keys.
package signer
