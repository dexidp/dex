# An overview of OpenID Connect

This document attempts to provide a general overview of the [OpenID Connect protocol](https://openid.net/connect/), a flavor of OAuth2 that dex implements. While this document isn't complete, we hope it provides enough information to get users up and running.

For an overview of custom claims, scopes, and client features implemented by dex, see [this document][scopes-claims-clients].

## OAuth2

OAuth2 should be familiar to anyone who's used something similar to a "Login
with Facebook" button. In these cases an application has chosen to let an
outside provider, in this case Facebook, attest to your identity instead of
having you set a username and password with the app itself.

The general flow for server side apps is:

1. A new user visits an application.
1. The application redirects the user to Facebook.
1. The user logs into Facebook, then is asked if it's okay to let the
application view the user's profile, post on their behalf, etc.
1. If the user clicks okay, Facebook redirects the user back to the application
with a code.
1. The application redeems that code with provider for a token that can be used
to access the authorized actions, such as viewing a users profile or posting on
their wall.

In these cases, dex is acting as Facebook (called the "provider" in OpenID
Connect) while clients apps redirect to it for the end user's identity. 

## ID Tokens

Unfortunately the access token applications get from OAuth2 providers is
completely opaque to the client and unique to the provider. The token you
receive from Facebook will be completely different from the one you'd get from
Twitter or GitHub.

OpenID Connect's primary extension of OAuth2 is an additional token returned in
the token response called the ID Token. This token is a [JSON Web Token](
https://tools.ietf.org/html/rfc7519) signed by the OpenID Connect server, with
well known fields for user ID, name, email, etc. A typical token response from
an OpenID Connect looks like (with less whitespace):

```
HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: no-store
Pragma: no-cache

{
 "access_token": "SlAV32hkKG",
 "token_type": "Bearer",
 "refresh_token": "8xLOxBtZp8",
 "expires_in": 3600,
 "id_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6IjFlOWdkazcifQ.ewogImlzc
   yI6ICJodHRwOi8vc2VydmVyLmV4YW1wbGUuY29tIiwKICJzdWIiOiAiMjQ4Mjg5
   NzYxMDAxIiwKICJhdWQiOiAiczZCaGRSa3F0MyIsCiAibm9uY2UiOiAibi0wUzZ
   fV3pBMk1qIiwKICJleHAiOiAxMzExMjgxOTcwLAogImlhdCI6IDEzMTEyODA5Nz
   AKfQ.ggW8hZ1EuVLuxNuuIJKX_V8a_OMXzR0EHR9R6jgdqrOOF4daGU96Sr_P6q
   Jp6IcmD3HP99Obi1PRs-cwh3LO-p146waJ8IhehcwL7F09JdijmBqkvPeB2T9CJ
   NqeGpe-gccMg4vfKjkM8FcGvnzZUN4_KSP0aAp1tOJ1zZwgjxqGByKHiOtX7Tpd
   QyHE5lcMiKPXfEIQILVq0pc_E2DzL7emopWoaoZTF_m0_N0YzFC6g6EJbOEoRoS
   K5hoDalrcvRYLSrQAZZKflyuVCyixEoV9GfNQC3_osjzw2PAithfubEEBLuVVk4
   XUVrWOLrLl0nx7RkKU8NXNHq-rvKMzqg"
}
```

That ID Token is a JWT with three base64'd fields separated by dots. The first
is a header, the second is a payload, and the third is a signature of the first
two fields. When parsed we can see the payload of this value is.

```
{
  "iss": "http://server.example.com",
  "sub": "248289761001",
  "aud": "s6BhdRkqt3",
  "nonce": "n-0S6_WzA2Mj",
  "exp": 1311281970,
  "iat": 1311280970
}
```

This has a few interesting fields such as

* The server that issued this token (`iss`).
* The token's subject (`sub`). In this case a unique ID of the end user.
* The token's audience (`aud`). The ID of the OAuth2 client this was issued for.

TODO: Add examples of payloads with "email" fields.

## Discovery

OpenID Connect servers have a discovery mechanism for OAuth2 endpoints, scopes
supported, and indications of various other OpenID Connect features.

```
$ curl http://127.0.0.1:5556/dex/.well-known/openid-configuration
{
  "issuer": "http://127.0.0.1:5556",
  "authorization_endpoint": "http://127.0.0.1:5556/auth",
  "token_endpoint": "http://127.0.0.1:5556/token",
  "jwks_uri": "http://127.0.0.1:5556/keys",
  "response_types_supported": [
    "code"
  ],
  "subject_types_supported": [
    "public"
  ],
  "id_token_signing_alg_values_supported": [
    "RS256"
  ],
  "scopes_supported": [
    "openid",
    "email",
    "profile"
  ]
}
```

Importantly, we've discovered the authorization endpoint, token endpoint, and
the location of the server's public keys. OAuth2 clients should be able to use
the token and auth endpoints immediately, while a JOSE library can be used to
parse the keys. The keys endpoint returns a [JSON Web Key](
https://tools.ietf.org/html/rfc7517) Set of public keys that will look
something like this:

```
$ curl http://127.0.0.1:5556/dex/keys
{
  "keys": [
    {
      "use": "sig",
      "kty": "RSA",
      "kid": "5d19a0fde5547960f4edaa1e1e8293e5534169ba",
      "alg": "RS256",
      "n": "5TAXCxkAQqHEqO0InP81z5F59PUzCe5ZNaDsD1SXzFe54BtXKn_V2a3K-BUNVliqMKhC2LByWLuI-A5ZlA5kXkbRFT05G0rusiM0rbkN2uvRmRCia4QlywE02xJKzeZV3KH6PldYqV_Jd06q1NV3WNqtcHN6MhnwRBfvkEIm7qWdPZ_mVK7vayfEnOCFRa7EZqr-U_X84T0-50wWkHTa0AfnyVvSMK1eKL-4yc26OWkmjh5ALfQFtnsz30Y2TOJdXtEfn35Y_882dNBDYBxtJV4PaSjXCxhiaIuBHp5uRS1INyMXCx2ve22ASNx_ERorv6BlXQoMDqaML2bSiN9N8Q",
      "e": "AQAB"
    }
  ]
}
```

[scopes-claims-clients]: custom-scopes-claims-clients.md
