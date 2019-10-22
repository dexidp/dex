package gocbcore

// UserPassPair represents a username and password pair.
type UserPassPair struct {
	Username string
	Password string
}

// AuthCredsRequest represents an authentication details request from the agent.
type AuthCredsRequest struct {
	Service  ServiceType
	Endpoint string
}

// AuthProvider is an interface to allow the agent to fetch authentication
// credentials on-demand from the application.
type AuthProvider interface {
	Credentials(req AuthCredsRequest) ([]UserPassPair, error)
}

func getSingleAuthCreds(auth AuthProvider, req AuthCredsRequest) (UserPassPair, error) {
	creds, err := auth.Credentials(req)
	if err != nil {
		return UserPassPair{}, err
	}

	if len(creds) != 1 {
		return UserPassPair{}, ErrInvalidCredentials
	}

	return creds[0], nil
}

func getKvAuthCreds(auth AuthProvider, endpoint string) (UserPassPair, error) {
	return getSingleAuthCreds(auth, AuthCredsRequest{
		Service:  MemdService,
		Endpoint: endpoint,
	})
}

func getMgmtAuthCreds(auth AuthProvider, endpoint string) (UserPassPair, error) {
	return getSingleAuthCreds(auth, AuthCredsRequest{
		Service:  MgmtService,
		Endpoint: endpoint,
	})
}

func getCapiAuthCreds(auth AuthProvider, endpoint string) (UserPassPair, error) {
	return getSingleAuthCreds(auth, AuthCredsRequest{
		Service:  CapiService,
		Endpoint: endpoint,
	})
}

func getN1qlAuthCreds(auth AuthProvider, endpoint string) ([]UserPassPair, error) {
	return auth.Credentials(AuthCredsRequest{
		Service:  N1qlService,
		Endpoint: endpoint,
	})
}

func getFtsAuthCreds(auth AuthProvider, endpoint string) ([]UserPassPair, error) {
	return auth.Credentials(AuthCredsRequest{
		Service:  FtsService,
		Endpoint: endpoint,
	})
}

func getCbasAuthCreds(auth AuthProvider, endpoint string) ([]UserPassPair, error) {
	return auth.Credentials(AuthCredsRequest{
		Service:  CbasService,
		Endpoint: endpoint,
	})
}

// PasswordAuthProvider provides a standard AuthProvider implementation
// for use with a standard username/password pair (for example, RBAC).
type PasswordAuthProvider struct {
	Username string
	Password string
}

// Credentials directly returns the username/password from the provider.
func (auth *PasswordAuthProvider) Credentials(req AuthCredsRequest) ([]UserPassPair, error) {
	return []UserPassPair{{
		Username: auth.Username,
		Password: auth.Password,
	}}, nil
}
