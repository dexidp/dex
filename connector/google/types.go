package google

type tokenClaims struct {
	Username      string `json:"name"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	HostedDomain  string `json:"hd"`
}
