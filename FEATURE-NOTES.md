Goal is to come up with a compact and minimal design for https://github.com/dexidp/dex/issues/32


# Notes

  - Use cookies to identify a returning user
  - Sign cookie using one of the internal private keys
  - Verify cookie when present
  - Generate and store cookie or cookie encrypted value on Store
    - Requires extension of storage.Storage interface
  - Feature Flag
  - Only introduce code in code-path of Password-based login providers
  - Write cookie to response on success login (see Ref(1))
  - Cookie ExpiresIn should be less or equal to the minted JWT ExpiresIn
  - I think simple store the storage.Claims+identity.ConnectorData and inject them into finalizeLogin, if the user is already logged in
  - An ActiveSession is only valid once it has an associated AccessToken
  - The ActiveSession should expire after a configurable amount of time



Ref(1):

probably here, and only in the case of http.MethodPost.

```go
func (s *Server) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
```
