# Examples

These are example uses of the oidc package. Each requires a Google account and the
client ID and secret of a registered OAuth2 application. The client ID and secret
should be set as the following environment variables:

```
GOOGLE_OAUTH2_CLIENT_ID
GOOGLE_OAUTH2_CLIENT_SECRET
```

See Google's documentation on how to set up an OAuth2 app:
https://developers.google.com/identity/protocols/OpenIDConnect?hl=en

Note that one of the redirect URL's must be "http://127.0.0.1:5556/auth/google/callback"
