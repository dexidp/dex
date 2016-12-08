# Authentication through SAML

## Overview

This connector adds basic support for receiving authentication data through SAML. It uses a fork of `github.com/RobotsAndPencils/go-saml` hosted at `github.com/Calpicow/go-saml`. There are some changes that haven't been merged into the main repo, and we could switch it back once they do.

## Configuration

The following is an example connector configuration for saml:

```yaml
connectors:
- type: saml
  id: saml
  name: saml
  config:
    issuerID: $ISSUER_ID
    signRequest: false
    # Typically urn:oasis:names:tc:SAML:2.0:nameid-format:transient or urn:oasis:names:tc:SAML:2.0:nameid-format:persistent
    nameIDPolicyFormat: urn:oasis:names:tc:SAML:2.0:nameid-format:persistent
    # Set to empty string to omit RequestedAuthnContext block
    requestedAuthnContext: urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport
    ssoURL: $SSO_URL
    idpPublicCertPath: /etc/dex/saml.crt
    idAttr: uid
    userAttr: cn
    emailAttr: mail
    groupsAttr: groups
    # Set to empty string if IDP returns groups as a separate AttributeValue
    groupsDelimiter: ','
    redirectURI: http://127.0.0.1:5556/callback
```
