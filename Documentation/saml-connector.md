# Authentication through SAML 2.0

## Overview

The SAML provider allows authentication through the SAML 2.0 HTTP POST binding. The connector maps attribute values in the SAML assertion to user info, such as username, email, and groups.

The connector uses the value of the `NameID` element as the user's unique identifier which dex assumes is both unique and never changes. Use the `nameIDPolicyFormat` to ensure this is set to a value which satisfies these requirements.

Unlike some clients which will process unprompted AuthnResponses, dex must send the initial AuthnRequest and validates the response's InResponseTo value.

## Caveats

__The connector doesn't support refresh tokens__ since the SAML 2.0 protocol doesn't provide a way to requery a provider without interaction. If the "offline_access" scope is requested, it will be ignored.

The connector doesn't support signed AuthnRequests or encrypted attributes.

## Configuration

```yaml
connectors:
- type: saml
  # Required field for connector id.
  id: saml
  # Required field for connector name.
  name: SAML
  config:
    # SSO URL used for POST value.
    ssoURL: https://saml.example.com/sso

    # CA to use when validating the signature of the SAML response.
    ca: /path/to/ca.pem

    # Dex's callback URL.
    #
    # If the response assertion status value contains a Destination element, it
    # must match this value exactly.
    #
    # This is also used as the expected audience for AudienceRestriction elements
    # if entityIssuer isn't specified.
    redirectURI: https://dex.example.com/callback

    # Name of attributes in the returned assertions to map to ID token claims.
    usernameAttr: name
    emailAttr: email
    groupsAttr: groups # optional

    # CA's can also be provided inline as a base64'd blob.
    #
    # caData: ( RAW base64'd PEM encoded CA )

    # To skip signature validation, uncomment the following field. This should
    # only be used during testing and may be removed in the future.
    #
    # insecureSkipSignatureValidation: true

    # Optional: Manually specify dex's Issuer value.
    #
    # When provided dex will include this as the Issuer value during AuthnRequest.
    # It will also override the redirectURI as the required audience when evaluating
    # AudienceRestriction elements in the response.
    entityIssuer: https://dex.example.com/callback

    # Optional: Issuer value expected in the SAML response.
    ssoIssuer: https://saml.example.com/sso

    # Optional: Delimiter for splitting groups returned as a single string.
    #
    # By default, multiple groups are assumed to be represented as multiple
    # attributes with the same name.
    #
    # If "groupsDelim" is provided groups are assumed to be represented as a
    # single attribute and the delimiter is used to split the attribute's value
    # into multiple groups.
    groupsDelim: ", "

    # Optional: Requested format of the NameID.
    #
    # The NameID value is is mapped to the user ID of the user. This can be an
    # abbreviated form of the full URI with just the last component. For example,
    # if this value is set to "emailAddress" the format will resolve to:
    #
    #     urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress
    #
    # If no value is specified, this value defaults to:
    #
    #     urn:oasis:names:tc:SAML:2.0:nameid-format:persistent
    #
    nameIDPolicyFormat: persistent
```

A minimal working configuration might look like:

```yaml
connectors:
- type: saml
  id: okta
  name: Okta
  config:
    ssoURL: https://dev-111102.oktapreview.com/app/foo/exk91cb99lKkKSYoy0h7/sso/saml
    ca: /etc/dex/saml-ca.pem
    redirectURI: http://127.0.0.1:5556/dex/callback
    usernameAttr: name
    emailAttr: email
    groupsAttr: groups
```
