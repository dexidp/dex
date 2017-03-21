# Authentication through SAML 2.0

## Overview

The experimental SAML provider allows authentication through the SAML 2.0 HTTP POST binding.

The connector uses the value of the `NameID` element as the user's unique identifier which dex assumes is both unique and never changes. Use the `nameIDPolicyFormat` to ensure this is set to a value which satisfies these requirements.

## Caveats

There are known issues with the XML signature validation for this connector. In addition work is still being done to ensure this connector implements best security practices for SAML 2.0.

The connector doesn't support signed AuthnRequests or encrypted attributes.

The connector doesn't support refresh tokens since the SAML 2.0 protocol doesn't provide a way to requery a provider without interaction. Ensure that the "offline_access" scope is not requested in client apps.

## Configuration

```yaml
connectors:
- type: samlExperimental # will be changed to "saml" later without support for the "samlExperimental" value
  # Required field for connector id.
  id: saml
  # Required field for connector name.
  name: SAML
  config:
    # SSO URL used for POST value.
    ssoURL: https://saml.example.com/sso

    # CA to use when validating the SAML response.
    ca: /path/to/ca.pem

    # CA's can also be provided inline as a base64'd blob. 
    #
    # caData: ( RAW base64'd PEM encoded CA )

    # To skip signature validation, uncomment the following field. This should
    # only be used during testing and may be removed in the future.
    # 
    # insucreSkipSignatureValidation: true

    # Dex's callback URL. Must match the "Destination" attribute of all responses
    # exactly.  
    redirectURI: https://dex.example.com/callback

    # Name of attributes in the returned assertions to map to ID token claims.
    usernameAttr: name
    emailAttr: email
    groupsAttr: groups # optional

    # By default, multiple groups are assumed to be represented as multiple
    # attributes with the same name.
    #
    # If "groupsDelim" is provided groups are assumed to be represented as a
    # single attribute and the delimiter is used to split the attribute's value
    # into multiple groups.
    #
    # groupsDelim: ", "


    # Requested format of the NameID. The NameID value is is mapped to the ID Token
    # 'sub' claim.  This can be an abbreviated form of the full URI with just the last
    # component. For example, if this value is set to "emailAddress" the format will
    # resolve to:
    #
    #     urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress
    #
    # If no value is specified, this value defaults to:
    #
    #     urn:oasis:names:tc:SAML:2.0:nameid-format:persistent
    #
    nameIDPolicyFormat: persistent

    # Optional issuer used for validating the SAML response. If provided the
    # connector will validate the Issuer in the response.
    # issuer: https://saml.example.com
```
