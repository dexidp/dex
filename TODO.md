TODOs in no particular order

OpenID Connect / OAuth2

- [ ] Let clients require signing algorithms (see id_token_signed_response_alg)
- [ ] Support ECDSA keys
- [ ] Support client_secret_jwt client authentication
- [ ] Add a "NextSigningKey" to the storage.Keys type so clients can cache more aggressively
- [ ] Support grant_type=password

Connectors

- [ ] Port BitBucket connector
- [ ] Port UAA connector
- [ ] Simplify LDAP connector configuration
- [ ] Create proposal for a minimal "local" connector implementation

User self-management

- [ ] Implement the user object proposal
- [ ] Provide user profile page
- [ ] Let user's merge accounts when they have multiple remote identities
- [ ] Let user's revoke clients with refresh tokens

Documentation

- [ ] Describe motivation for a V2
- [ ] Add OpenID Connect client library suggestions
- [ ] Add getting started guide
- [ ] Add more connector documentation
  - [ ] Include instructions for getting client credentials for upstream provider
- [ ] Improve Kubernetes documentation and include client auth provider docs

Storage

- [x] Add SQL storage implementation
- [ ] Utilize fixes for third party resources in Kubernetes 1.4 

UX

- [ ] Add 500 and 404 pages
- [ ] Add an OBB template
- [ ] Set an HTTP cookie so users aren't constantly reprompted for passwords
- [ ] Add proposal for letting others style existing HTML templates
- [ ] Support serving arbitrary static assets

Backend

- [ ] Improve logging, possibly switch to logrus
- [ ] Standardize OAuth2 error handling
- [ ] Switch to github.com/ghodss/yaml for []byte to base64 string logic
