# OIDC Connect Core Notes

dex aims to be a full featured OIDC Provider, but it still has a little ways to go. Most of the places where dex and OIDC diverge are minor, but we do want to fix them. Here's a list of these places; there may be other discrepancies as well if you find any please file an issue or even better, a pull request.

To be clear: the places were we are not in compliance with mandatory features, we will fix. As for things marked as OPTIONAL in the spec, whether and when those are supported by dex will be driven by the needs of the community.

# Notes on [OpenID Connect Core](http://openid.net/specs/openid-connect-core-1_0.html)


Sec. 2. [ID Token](http://openid.net/specs/openid-connect-core-1_0.html#IDToken)
- None of the OPTIONAL claims  (`acr`, `amr`, `azp`, `auth_time`) are supported
- dex signs using JWS but does not do the OPTIONAL encryption.


Sec. 3. [Authentication](http://openid.net/specs/openid-connect-core-1_0.html#Authentication)
- Only the authorization code flow (where `response_type` is `code`) is supported.

Sec. 3.1.2. [Authorization Endpoint](http://openid.net/specs/openid-connect-core-1_0.html#AuthorizationEndpoint)
- In a production system TLS is required but the dex web-server only supports HTTP right now - it is expected that until HTTPS is supported, TLS termination will be handled outside of dex.


Sec. 3.1.2.1. [Authentication Request](http://openid.net/specs/openid-connect-core-1_0.html#AuthRequest)
- dex doesn't check the value of "scope" to make sure it contains "openid" in authentication requests.
- max_age not implemented; it's OPTIONAL in the spec, but if it's present servers MUST include auth_time, which dex does not.
- None of the other OPTIONAL parameters are implemented with the exception of:
  - state
  - nonce
- dex also defines a non-standard `register` parameter; when this parameter is `1`, end-users are taken through a registration flow, which after completing successfully, lands them at the specified `redirect_uri`

Sec. 3.1.2.2. [Authentication Request Validation](http://openid.net/specs/openid-connect-core-1_0.html#AuthRequestValidation)
- As mentioned earlier, dex doesn't validate that the `openid` scope value is present.


Sec. 3.2.2.3. [Authorization Server Authenticates End-User](http://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthenticates)
- The spec states that the authentication server "MUST NOT interact with the End-User" when `prompt` is `none` We don't check the prompt parameter at all; similarly, dex MUST re-prompt when `prompt` is `login` - dex does not do this either.


Sec. 3.1.3.2. [Token Request Validation](http://openid.net/specs/openid-connect-core-1_0.html#TokenRequestValidation)
- In Token requests, dex chooses to proceed without error when `redirect_uri` is not present and there's only one registered valid URI (which is valid behavior)

Sec. 4.  [Initiating Login from a Third Party](http://openid.net/specs/openid-connect-core-1_0.html#ThirdPartyInitiatedLogin)
    - dex does not support this at this time

Sec. 5.1.2. [AdditionalClaims](http://openid.net/specs/openid-connect-core-1_0.html#AdditionalClaims)
- dex defines uses the following additional claims:
  - `http://coreos.com/password/old-hash`
  - `http://coreos.com/password/reset-callback`
  - `http://coreos.com/email/verification-callback`
  - `http://coreos.com/email/verificationEmail`

Sec. 5.3.  [UserInfo Endpoint](http://openid.net/specs/openid-connect-core-1_0.html#UserInfo)
- dex does not implement this endpoint.

Sec. 6.1 [Passing a Request Object by Value](http://openid.net/specs/openid-connect-core-1_0.html#JWTRequests)
- dex does not implement this feature.

Sec. 7. [Self-Issued OpenID Provider](http://openid.net/specs/openid-connect-core-1_0.html#SelfIssued)
- dex does not implement this feature.

Sec. 8. [Subject Identifier Types](http://openid.net/specs/openid-connect-core-1_0.html#SubjectIDTypes)
- dex only supports the `public` subject identifier type.

Sec. 9. [Client Authentication](http://openid.net/specs/openid-connect-core-1_0.html#ClientAuthentication)
- dex only supports the `client_secret_basic` client authentication type.

Sec. 11. [Offline Access](http://openid.net/specs/openid-connect-core-1_0.html#OfflineAccess)
- dex does not implement this feature.


Sec. 15.1.  [Mandatory to Implement Features for All OpenID Providers](http://openid.net/specs/openid-connect-core-1_0.html#ImplementationConsiderations)
- dex is missing the follow mandatory features (some are already noted elsewhere in this document):
  - Support for `prompt` parameter
  - Support for the `auth_time` parameter
  - Support for enforcing `max_age` parameter

Sec. 15.3. [Discovery and Registration](http://openid.net/specs/openid-connect-core-1_0.html#DiscoReg)
- dex supports OIDC Discovery at the standard `/.well-known/openid-configuration` endpoint.
