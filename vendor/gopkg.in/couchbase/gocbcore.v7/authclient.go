package gocbcore

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"time"
)

// TODO(brett19): Remove the Exec keyword from AuthClient

// AuthClient exposes an interface for performing authentication on a
// connected Couchbase K/V client.
type AuthClient interface {
	Address() string
	SupportsFeature(feature HelloFeature) bool

	ExecSaslListMechs(deadline time.Time) ([]string, error)
	ExecSaslAuth(k, v []byte, deadline time.Time) ([]byte, error)
	ExecSaslStep(k, v []byte, deadline time.Time) ([]byte, error)
	ExecSelectBucket(b []byte, deadline time.Time) error
}

// SaslAuthPlain performs PLAIN SASL authentication against an AuthClient.
func SaslAuthPlain(username, password string, client AuthClient, deadline time.Time) error {
	// Build PLAIN auth data
	userBuf := []byte(username)
	passBuf := []byte(password)
	authData := make([]byte, 1+len(userBuf)+1+len(passBuf))
	authData[0] = 0
	copy(authData[1:], userBuf)
	authData[1+len(userBuf)] = 0
	copy(authData[1+len(userBuf)+1:], passBuf)

	// Execute PLAIN authentication
	_, err := client.ExecSaslAuth([]byte("PLAIN"), authData, deadline)

	return err
}

func saslAuthScram(saslName []byte, newHash func() hash.Hash, username, password string, client AuthClient, deadline time.Time) error {
	scramMgr := newScramClient(newHash, username, password)

	var in []byte
	var err error

	// Perform the initial SASL step
	scramMgr.Step(nil)
	in, err = client.ExecSaslAuth(saslName, scramMgr.Out(), deadline)
	if err != nil && !IsErrorStatus(err, StatusAuthContinue) {
		return err
	}

	// Perform any additional step rounds
	for IsErrorStatus(err, StatusAuthContinue) {
		if !scramMgr.Step(in) {
			err = scramMgr.Err()
			if err != nil {
				return err
			}

			logErrorf("Local auth client finished before server accepted auth")
			return ErrAuthError
		}

		in, err = client.ExecSaslStep(saslName, scramMgr.Out(), deadline)
		if err != nil &&
			!IsErrorStatus(err, StatusAuthContinue) {
			return err
		}
	}

	return nil
}

// SaslAuthScramSha1 performs SCRAM-SHA1 SASL authentication against an AuthClient.
func SaslAuthScramSha1(username, password string, client AuthClient, deadline time.Time) error {
	return saslAuthScram([]byte("SCRAM-SHA1"), sha1.New, username, password, client, deadline)
}

// SaslAuthScramSha256 performs SCRAM-SHA256 SASL authentication against an AuthClient.
func SaslAuthScramSha256(username, password string, client AuthClient, deadline time.Time) error {
	return saslAuthScram([]byte("SCRAM-SHA256"), sha256.New, username, password, client, deadline)
}

// SaslAuthScramSha512 performs SCRAM-SHA512 SASL authentication against an AuthClient.
func SaslAuthScramSha512(username, password string, client AuthClient, deadline time.Time) error {
	return saslAuthScram([]byte("SCRAM-SHA512"), sha512.New, username, password, client, deadline)
}

// SaslAuthBest performs SASL authentication against an AuthClient using the
// best supported authentication algorithm available on both client and server.
func SaslAuthBest(username, password string, client AuthClient, deadline time.Time) error {
	methods, err := client.ExecSaslListMechs(deadline)
	if err != nil {
		return err
	}

	logDebugf("Server SASL supports: %v", methods)

	var bestMethod string
	var bestPriority int
	for _, method := range methods {
		if bestPriority <= 1 && method == "PLAIN" {
			bestPriority = 1
			bestMethod = method
		}

		// CRAM-MD5 is intentionally disabled here as it provides a false
		// sense of security to users.  SCRAM-SHA1 should be used, or TLS
		// connection coupled with PLAIN auth would also be sufficient.
		/*
			if bestPriority <= 2 && method == "CRAM-MD5" {
				bestPriority = 2
				bestMethod = method
			}
		*/

		if bestPriority <= 3 && method == "SCRAM-SHA1" {
			bestPriority = 3
			bestMethod = method
		}

		if bestPriority <= 4 && method == "SCRAM-SHA256" {
			bestPriority = 4
			bestMethod = method
		}

		if bestPriority <= 5 && method == "SCRAM-SHA512" {
			bestPriority = 5
			bestMethod = method
		}
	}

	logDebugf("Selected `%s` for SASL auth", bestMethod)

	if bestMethod == "PLAIN" {
		return SaslAuthPlain(username, password, client, deadline)
	} else if bestMethod == "SCRAM-SHA1" {
		return SaslAuthScramSha1(username, password, client, deadline)
	} else if bestMethod == "SCRAM-SHA256" {
		return SaslAuthScramSha256(username, password, client, deadline)
	} else if bestMethod == "SCRAM-SHA512" {
		return SaslAuthScramSha512(username, password, client, deadline)
	}

	return ErrNoAuthMethod
}
