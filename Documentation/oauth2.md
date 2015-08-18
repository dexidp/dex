# dex OAuth 2.0 Implementation

OAuth 2.0 is defined in [RFC 6749][rfc6749]. The RFC defines the bare minimum necessary to implement OAuth 2.0, while it also describes a set of optional behaviors. This document aims to describe what decisions have been made in dex with respect to those optional behaviors.

[rfc6749]: https://tools.ietf.org/html/rfc6749

While the goal of dex is to accurately implement the required aspects of the OAuth 2.0 specification, dex is still under active development and certain disrepancies exist. Any such discrepancies are documented below.

## TLS

It is a requirement of OAuth 2.0 (RFC 6749 Section 2.3.1) that any authorization server utilize TLS to protect sensitive information (e.g. client passwords) transmitted between remote parties during the OAuth workflow.
From a practical standpoint, TLS can be a tedious requirement for development environments.
It also may be desirable to deploy dex behind an SSL-terminating load balancer.
dex does not require the use of TLS, but it should be considered necessary when deploying dex on any public networks.

## Client Authentication

Unregistered clients are not supported (RFC 6749 Section 2.4).

## Authorization Endpoint

User-agent MUST make a valid authorization request to the authorization endpoint (RFC 6749 Section 3.1) using the "authorization code" grant type; no other grant types are supported.
Additionally, GET w/ query parameters must be used - POST w/ a form in the request body body is not supported.


The OAuth 2.0 spec leaves the implementation details of the authorization step up to the implementer as long as the request and response formats are respected.
dex relies on remote identity providers to fulfill the actual authorization of user-agents.
This federated approach typically requires one or more additional HTTP redirects and a manual login step past what is encountered with a typical OAuth 2.0 server.
Given this detail, it is necessary that user-agents follow all HTTP redirects during an authorization attempt.
Additionally, the HTTP response from the initial authorization request will likely not redirect the user-agent to the redirection endpoint provided in that initial request.
User-agent MUST not reject redirections to unrecognized endpoints.

## Token endpoint

Clients MUST identify themselves using the Basic HTTP authentication scheme (RFC 6749 Section 2.3.1).
Given this requirement, the client_id and client_secret fields of the request are ignored.

Refresh tokens are never generated and returned.

Given that the authorization endpoint only supports authorization codes and refresh tokens are never generated, the only supported values of grant_type are "authorization_code" and "client_credentials".


