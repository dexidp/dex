/*
Package oidc implements OpenID Connect client logic for the golang.org/x/oauth2 package.

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

OAuth2 redirects are unchanged.

	func handleRedirect(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, oauth2Config.AuthCodeURL(state), http.StatusFound)
	})

For callbacks the provider can be used to query for user information such as email.

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

The provider also has the ability to verify ID Tokens.

	verifier := provider.NewVerifier(ctx)

The returned verifier can be used to perform basic validation on ID Token issued by the provider,
including verifying the JWT signature. It then returns the payload.

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
			http.Error(w, "Failed to unmarshal ID Token custom claims: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// ...
	})

ID Token nonces are supported.

First, provide a nonce source for nonce validation. This will then be used to wrap the existing
provider ID Token verifier.

	// A verifier which boths verifies the ID Token signature and nonce.
	nonceEnabledVerifier := provider.NewVerifier(ctx, oidc.VerifyNonce(nonceSource))

For the redirect provide a nonce auth code option. This will be placed as a URL parameter during
the client redirect.

	func handleRedirect(w http.ResponseWriter, r *http.Request) {
		nonce, err := newNonce()
		if err != nil {
			// ...
		}
		// Provide a nonce for the OpenID Connect ID Token.
		http.Redirect(w, r, oauth2Config.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)
	})

The nonce enabled verifier can then be used to verify the nonce while unpacking the ID Token.

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

		// Verify that the ID Token is signed by the provider and verify the nonce.
		idToken, err := nonceEnabledVerifier.Verify(rawIDToken)
		if err != nil {
			http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Continue as above...
	})

This package uses contexts to derive HTTP clients in the same way as the oauth2 package. To configure
a custom client, use the oauth2 packages HTTPClient context key when constructing the context.

	myClient := &http.Client{}

	myCtx := context.WithValue(parentCtx, oauth2.HTTPClient, myClient)

	// NewProvider will use myClient to make the request.
	provider, err := oidc.NewProvider(myCtx, "https://accounts.example.com")
*/
package oidc
