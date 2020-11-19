# SSO

## Intro

Single sign-on (SSO) is an authentication scheme that allows a user to log in with a single ID and password to any of several related, yet independent, software systems.
True single sign on allows the user to login once and access services without re-entering authentication factors.

In our case, to use the SSO scheme, we will use the session way with the golang web toolkit, **gorilla**

## Dex Modification

All added code are stored into a single file: **./server/sso.go**

### Authentification of the User for the first time

Process of Dex workflow. After the redirect to Dexd from an IDP, we catch the **identity struct** encrypted and store it into the session

```go
session := s.getSession(r, authReq)
session.Values["identity"], err = json.Marshal(identity)
session.Save(r, w)
```

Then the Approval page is shown to the Client and we store all **Scopes struct** wanted by the Client into the same session

```go
session := s.getSession(r, authReq)
session.Values["scopes"], err = json.Marshal(scopes)
session.Save(r, w)
```

### Authentification of the User for the second time

- Process of Dex workflow up to IDP chosen page. User will chose the IDP (LDAP, Gitlab, Github, ...).
- We search into the session store a session with the ID of the IDP name and take the identity struct from it
- we connect with this struct from the IDP, **to loggin authomatically** the User

## TODO

For now the SSO is working, but the sessionStore.Options.Secure is set to false