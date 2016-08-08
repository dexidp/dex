# OpenID Connect client support for Go

[![GoDoc](https://godoc.org/github.com/ericchiang/oidc?status.svg)](https://godoc.org/github.com/ericchiang/oidc)

This package implements OpenID Connect client logic for the golang.org/x/oauth2 package.

```go
provider, err := oidc.NewProvider(ctx, "https://accounts.example.com")
if err != nil {
	return err
}

// Configure an OpenID Connect aware OAuth2 client.
oauth2Config := oauth2.Config{
	ClientID:     clientID,
	ClientSecret: clientSecret,
	RedirectURL:  redirectURL,
	Endpoint:     provider.Endpoint(),
	Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
}
```

OAuth2 redirects are unchanged.

```go
func handleRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, oauth2Config.AuthCodeURL(state), http.StatusFound)
})
```

For callbacks the provider can be used to query for [user information](https://openid.net/specs/openid-connect-core-1_0.html#UserInfo) such as email.

```go
func handleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	// Verify state...

	oauth2Token, err := oauth2Config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userinfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		http.Error(w, "Failed to get userinfo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// ...
})
```

Or the provider can be used to verify and inspect the OpenID Connect
[ID Token](https://openid.net/specs/openid-connect-core-1_0.html#IDToken) in the
[token response](https://openid.net/specs/openid-connect-core-1_0.html#TokenResponse).

```go
verifier := provider.NewVerifier(ctx)
```

The verifier itself can be constructed with addition checks, such as verifing a
token was issued for a specific client or hasn't expired.

```go
verifier := provier.NewVerifier(ctx, oidc.VerifyAudience(clientID), oidc.VerifyExpiry())
```

The returned verifier can be used to ensure the ID Token (a JWT) is signed by the provider. 

```go
func handleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	// Verify state...

	oauth2Token, err := oauth2Config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract the ID Token from oauth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No ID Token found", http.StatusInternalServerError)
		return
	}

	// Verify that the ID Token is signed by the provider.
	idToken, err := verifier.Verify(rawIDToken)
	if err != nil {
		http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Unmarshal ID Token for expected custom claims.
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "Failed to unmarshal ID Token claims: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// ...
})
```
