/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package k8sapi

// Where possible, json tags match the cli argument names.
// Top level config objects and all values required for proper functioning are not "omitempty".  Any truly optional piece of config is allowed to be omitted.

// Config holds the information needed to build connect to remote kubernetes clusters as a given user
type Config struct {
	// Legacy field from pkg/api/types.go TypeMeta.
	// TODO(jlowdermilk): remove this after eliminating downstream dependencies.
	Kind string `yaml:"kind,omitempty"`
	// DEPRECATED: APIVersion is the preferred api version for communicating with the kubernetes cluster (v1, v2, etc).
	// Because a cluster can run multiple API groups and potentially multiple versions of each, it no longer makes sense to specify
	// a single value for the cluster version.
	// This field isn't really needed anyway, so we are deprecating it without replacement.
	// It will be ignored if it is present.
	APIVersion string `yaml:"apiVersion,omitempty"`
	// Preferences holds general information to be use for cli interactions
	Preferences Preferences `yaml:"preferences"`
	// Clusters is a map of referencable names to cluster configs
	Clusters []NamedCluster `yaml:"clusters"`
	// AuthInfos is a map of referencable names to user configs
	AuthInfos []NamedAuthInfo `yaml:"users"`
	// Contexts is a map of referencable names to context configs
	Contexts []NamedContext `yaml:"contexts"`
	// CurrentContext is the name of the context that you would like to use by default
	CurrentContext string `yaml:"current-context"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	Extensions []NamedExtension `yaml:"extensions,omitempty"`
}

// Preferences contains information about the users command line experience preferences.
type Preferences struct {
	Colors bool `yaml:"colors,omitempty"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	Extensions []NamedExtension `yaml:"extensions,omitempty"`
}

// Cluster contains information about how to communicate with a kubernetes cluster
type Cluster struct {
	// Server is the address of the kubernetes cluster (https://hostname:port).
	Server string `yaml:"server"`
	// APIVersion is the preferred api version for communicating with the kubernetes cluster (v1, v2, etc).
	APIVersion string `yaml:"api-version,omitempty"`
	// InsecureSkipTLSVerify skips the validity check for the server's certificate. This will make your HTTPS connections insecure.
	InsecureSkipTLSVerify bool `yaml:"insecure-skip-tls-verify,omitempty"`
	// CertificateAuthority is the path to a cert file for the certificate authority.
	CertificateAuthority string `yaml:"certificate-authority,omitempty"`
	// CertificateAuthorityData contains PEM-encoded certificate authority certificates. Overrides CertificateAuthority
	//
	// NOTE(ericchiang): Our yaml parser doesn't assume []byte is a base64 encoded string.
	CertificateAuthorityData string `yaml:"certificate-authority-data,omitempty"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	Extensions []NamedExtension `yaml:"extensions,omitempty"`
}

// AuthInfo contains information that describes identity information.  This is use to tell the kubernetes cluster who you are.
type AuthInfo struct {
	// ClientCertificate is the path to a client cert file for TLS.
	ClientCertificate string `yaml:"client-certificate,omitempty"`
	// ClientCertificateData contains PEM-encoded data from a client cert file for TLS. Overrides ClientCertificate
	//
	// NOTE(ericchiang): Our yaml parser doesn't assume []byte is a base64 encoded string.
	ClientCertificateData string `yaml:"client-certificate-data,omitempty"`
	// ClientKey is the path to a client key file for TLS.
	ClientKey string `yaml:"client-key,omitempty"`
	// ClientKeyData contains PEM-encoded data from a client key file for TLS. Overrides ClientKey
	//
	// NOTE(ericchiang): Our yaml parser doesn't assume []byte is a base64 encoded string.
	ClientKeyData string `yaml:"client-key-data,omitempty"`
	// Token is the bearer token for authentication to the kubernetes cluster.
	Token string `yaml:"token,omitempty"`
	// Impersonate is the username to imperonate.  The name matches the flag.
	Impersonate string `yaml:"as,omitempty"`
	// Username is the username for basic authentication to the kubernetes cluster.
	Username string `yaml:"username,omitempty"`
	// Password is the password for basic authentication to the kubernetes cluster.
	Password string `yaml:"password,omitempty"`
	// AuthProvider specifies a custom authentication plugin for the kubernetes cluster.
	AuthProvider *AuthProviderConfig `yaml:"auth-provider,omitempty"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	Extensions []NamedExtension `yaml:"extensions,omitempty"`
}

// Context is a tuple of references to a cluster (how do I communicate with a kubernetes cluster), a user (how do I identify myself), and a namespace (what subset of resources do I want to work with)
type Context struct {
	// Cluster is the name of the cluster for this context
	Cluster string `yaml:"cluster"`
	// AuthInfo is the name of the authInfo for this context
	AuthInfo string `yaml:"user"`
	// Namespace is the default namespace to use on unspecified requests
	Namespace string `yaml:"namespace,omitempty"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	Extensions []NamedExtension `yaml:"extensions,omitempty"`
}

// NamedCluster relates nicknames to cluster information
type NamedCluster struct {
	// Name is the nickname for this Cluster
	Name string `yaml:"name"`
	// Cluster holds the cluster information
	Cluster Cluster `yaml:"cluster"`
}

// NamedContext relates nicknames to context information
type NamedContext struct {
	// Name is the nickname for this Context
	Name string `yaml:"name"`
	// Context holds the context information
	Context Context `yaml:"context"`
}

// NamedAuthInfo relates nicknames to auth information
type NamedAuthInfo struct {
	// Name is the nickname for this AuthInfo
	Name string `yaml:"name"`
	// AuthInfo holds the auth information
	AuthInfo AuthInfo `yaml:"user"`
}

// NamedExtension relates nicknames to extension information
type NamedExtension struct {
	// Name is the nickname for this Extension
	Name string `yaml:"name"`
}

// AuthProviderConfig holds the configuration for a specified auth provider.
type AuthProviderConfig struct {
	Name   string            `yaml:"name"`
	Config map[string]string `yaml:"config"`
}
