package gocb

import (
	"gopkg.in/couchbase/gocbcore.v7"
)

// UserPassPair represents a username and password pair.
type UserPassPair gocbcore.UserPassPair

type coreAuthWrapper struct {
	auth       Authenticator
	bucketName string
}

// Credentials returns the credentials for a particular service.
func (auth *coreAuthWrapper) Credentials(req gocbcore.AuthCredsRequest) ([]gocbcore.UserPassPair, error) {
	creds, err := auth.auth.Credentials(AuthCredsRequest{
		Service:  ServiceType(req.Service),
		Endpoint: req.Endpoint,
		Bucket:   auth.bucketName,
	})
	if err != nil {
		return nil, err
	}

	coreCreds := make([]gocbcore.UserPassPair, len(creds))
	for credIdx, userPass := range creds {
		coreCreds[credIdx] = gocbcore.UserPassPair(userPass)
	}
	return coreCreds, nil
}

// AuthCredsRequest encapsulates the data for a credential request
// from the new Authenticator interface.
// UNCOMMITTED
type AuthCredsRequest struct {
	Service  ServiceType
	Endpoint string
	Bucket   string
}

func getSingleCredential(auth Authenticator, req AuthCredsRequest) (UserPassPair, error) {
	creds, err := auth.Credentials(req)
	if err != nil {
		return UserPassPair{}, err
	}

	if len(creds) != 1 {
		return UserPassPair{}, gocbcore.ErrInvalidCredentials
	}

	return creds[0], nil
}

// Authenticator provides an interface to authenticate to each service.  Note that
// only authenticators implemented here are stable, and support for custom
// authenticators is considered volatile.
type Authenticator interface {
	Credentials(req AuthCredsRequest) ([]UserPassPair, error)
}

// BucketAuthenticator provides a password for a single bucket.
type BucketAuthenticator struct {
	Password string
}

// BucketAuthenticatorMap is a map of bucket name to BucketAuthenticator.
type BucketAuthenticatorMap map[string]BucketAuthenticator

// ClusterAuthenticator implements an Authenticator which uses a list of buckets and passwords.
type ClusterAuthenticator struct {
	Buckets  BucketAuthenticatorMap
	Username string
	Password string
}

func (ca ClusterAuthenticator) clusterCreds() []UserPassPair {
	var creds []UserPassPair
	for bucketName, bucket := range ca.Buckets {
		creds = append(creds, UserPassPair{
			Username: bucketName,
			Password: bucket.Password,
		})
	}
	return creds
}

// Credentials returns the credentials for a particular service.
func (ca ClusterAuthenticator) Credentials(req AuthCredsRequest) ([]UserPassPair, error) {
	if req.Bucket == "" {
		if req.Service == MemdService || req.Service == MgmtService ||
			req.Service == CapiService {
			return []UserPassPair{{
				Username: ca.Username,
				Password: ca.Password,
			}}, nil
		}

		return ca.clusterCreds(), nil
	}

	if bucketAuth, ok := ca.Buckets[req.Bucket]; ok {
		return []UserPassPair{{
			Username: req.Bucket,
			Password: bucketAuth.Password,
		}}, nil
	}

	return []UserPassPair{{
		Username: "",
		Password: "",
	}}, nil
}

// PasswordAuthenticator implements an Authenticator which uses an RBAC username and password.
type PasswordAuthenticator struct {
	Username string
	Password string
}

// Credentials returns the credentials for a particular service.
func (ra PasswordAuthenticator) Credentials(req AuthCredsRequest) ([]UserPassPair, error) {
	return []UserPassPair{{
		Username: ra.Username,
		Password: ra.Password,
	}}, nil
}

type certAuthenticator interface {
	isTlsAuth() bool
}

// CertAuthenticator implements an Authenticator which can be used with certificate authentication.
type CertAuthenticator struct {
}

// Credentials returns the credentials for a particular service.
func (ca CertAuthenticator) Credentials(req AuthCredsRequest) ([]UserPassPair, error) {
	return []UserPassPair{{
		Username: "",
		Password: "",
	}}, nil
}

func (ca CertAuthenticator) isTlsAuth() bool {
	return true
}

// CertificateAuthenticator is included for backwards compatibility only.
// Deprecated: Use CertAuthenticator instead.
type CertificateAuthenticator struct {
	CertAuthenticator
}

func (ca CertificateAuthenticator) isTlsAuth() bool {
	return true
}
