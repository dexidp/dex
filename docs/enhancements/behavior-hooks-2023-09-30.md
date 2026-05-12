# Dex Enhancement Proposal (DEP) - 2023-09-30 - Behavior Hooks

## Summary

In this document, we would like to propose a generic mechanism to inject custom logic into Dex's logic without increasing the complexity of Dex's codebase.

## Context

Customizing the authentication flow in an OIDC provider like Dex may be useful for several reasons: customizing the authentication flow, adding custom claims to the ID token, validating the ID token claims, and many others.

Customizing the authentication flow is a very common feature amongst multiple OIDC-related providers:
* https://zitadel.com/blog/custom-claims
* https://developer.okta.com/docs/guides/customize-tokens-returned-from-okta/main/#request-a-token-that-contains-the-custom-claim

To accomplish this, many proposals were made to add this feature to Dex:

- https://github.com/dexidp/dex/issues/2578
- https://github.com/dexidp/dex/issues/2778
- https://github.com/dexidp/dex/issues/2838
- https://github.com/dexidp/dex/issues/2836
- https://github.com/dexidp/dex/pull/2963
- https://github.com/dexidp/dex/pull/2960

Many of them are trying to introduce some kind of customization to the connector/claims logic, but they remove the genericity of dex by adding some use-case-specific code.

## Motivation

Dex has a very rigid logic support. By design, the logic is kept minimal so as not to increase the codebase's overall complexity. The only way to extend Dex's logic is to fork Dex and its custom behavior. Many PRs have been proposed to add specific workflows or changes to the Dex codebase.
We would like to add a generic mechanism to inject custom logic into Dex's logic without increasing the complexity of Dex's codebase.

### Goals/Pain

This document aims to define a maintainable, secure, and extensible mechanism to inject the custom logic into Dex's logic.

This includes two aspects:

* *Connectors*: customizing and filtering the connectors list based on the request context.
* *OIDC Token Claims*: adding/removing/modifying the OIDC token claims based on the Connector and the request context.

This document aims to propose and agree on a generic mechanism to inject custom logic into Dex's logic. The principles defined in this document will be better refined and implemented by dedicated PR.

### Non-goals

* Discuss specific hooks implementations.
* Introduce any use-case-specific logic in the Dex code.

## Proposal

We propose to introduce 2 *types* to hook the Dex behavior:

* **Internal** is defined in the same process as Dex and should use some computationally bounded, sandboxed language, such as Cel or Rego. This approach is more secure and performant.
* **External** are defined in a separate process and communicate with Dex over gRPC/Rest. This approach would be more flexible but less secure and performant.

We propose to introduce 2 *resources* that can be hooked:

* *Connectors*: introducing the possibility to filter/modify the connectors list based on the request context.
* *ID Token Claims*: introducing the possibility to Add/Remove/Modify/Validate the ID token claims based on the request context.

### Connectors

Connectors are used to authenticate users. The connectors are chosen when a user tries to authenticate. The entire list of connectors is defined in the dex config file.
Sometimes, filtering the connectors based on the request context and connectors properties is convenient. For example, we would like to show different connectors based on the user's browser, custom parameters, etc.

The workflow of a request in the presence of connectors filters is as follows:

1. Dex receives a request to authenticate.
2. Dex receives the requests, compiles the request context and the list of connectors, and sends it to the list of hooks.
3. Each hook receives the list of connectors and the request context, applies its filter, and returns a new list.

Eventually, the returned list of connectors is shown to the user.

When a user tries to authenticate, it performs a request to Dex. The request contains a set of **parameters** and **headers** used to filter.

The request context is defined in the dex config file as follows:

```go
type RequestContext struct {
    Headers map[string]string
    Params map[string]string
}
```

Connectors have a set of properties that can be used to filter the connectors list. For example, the connector type, the connector name, etc.
However, they often contain credentials and configurations that should not be exposed to the hooks. For example, the client secret of the OAuth2 connector.

Therefore, the `ConnectorContext` is defined as follows:

```go
type ConnectorContext struct {
    Type string
    Name string
    ID string
    // ...
}
```

The interface implemented by the filter would be defined as follows:

```go
type ConnectorWebhookFilter interface {
  FilterConnectors(connectors []ConnectorContext, r requestContext) ([]ConnectorContext, error)
}
```

The result of the `FilterConnectors` method is a list of connectors that should be shown to the user or an empty list with an error in case of failure.

### ID Token Claims

ID Token claims are used to add information about the user to the ID token. They are typically used to add information about the user's groups, email, etc.
The claims are added to the ID token when the user authenticates and are used by the client to authorize the user to access some resources.

So far, Dex does not provide a way to add/remove/modify the ID token claims based on the request context. This prevents the possibility of contextualizing the ID token claims based on the request context (i.e., the connector) used to authenticate the user).

We propose to introduce a mechanism to add/remove/modify the ID token claims based on the request context used to authenticate the user. Such a mechanism will allow the possibility to implement advanced behaviors (i.e., claims prefixing) in a single Dex instance without hardcoding any logic in Dex.

The workflow of a request in the presence of ID Token filters is as follows:

1. Upon a successful request to authenticate, Dex compiles the new ID token.
2. When the token is fully compiled, Dex sends ID token claims context and the Connector ID to the hooks.
3. Each hook receives the ID token claims context and the Connector ID, applies its filter, and returns a new list.

We identify two categories of ID token claims webhooks:

* Mutating: add/remove/modify the ID token claims based on the request context. The `MutateIDTokenClaims` method is used to mutate the ID token claims. The method returns the mutated claims.
* Validating: validate the ID token claims based on the request context. The `ValidateIDTokenClaims` method is used to validate the ID token claims. If the method returns a bool. If false, the request is rejected.

The hooks are always executed following the order defined in the dex config file and with the validation hooks executed after the mutation hooks to allow the validation hooks to validate the claims after the mutation hooks.

```go
type IDTokenClaimsWebhook interface {
    ValidateIDTokenClaims(claims map[string]interface{}, connID string) bool
    MutateIDTokenClaims(claims map[string]interface{},connID string) (map[string]interface{}, error)
}
```

### User Experience

#### Connectors

The connector filters would be defined in the dex config file as follows:

```yaml
connectorFilters:
    - name: "filter1"
      type: "internal"
      requestContext:
        headers:
        - "X-Forwarded-For"
        - "X-Forwarded-Proto"
        params:
        - "country"
      config:
        # ...
    - name: "filter2"
      type: "external"
      requestContext:
        headers:
        - "X-Forwarded-For"
        - "X-Forwarded-Proto"
        params:
        - "country"
      config:
        # ...
```

Each filter is executed in the order defined in the dex config file. A filter is characterized by:

* *name*: the name of the filter.
* *type*: the type of the filter. It can be `internal` or `external`.
* *request Context*: the request context used to filter the connectors.
* *config*: the configuration of the filter. For the internal filters, it is the Cel/Rego expression. For the external filters, it would contain the parameters of the webhook (e.g., URL, caBundle, token).

The ID token claims webhooks are defined in the dex config file as follows:

```yaml
tokenClaimsHooks:
  validatingHooks:
    - name: "Validation 1"
      type: "internal"
      failurePolicy: Fail
      claims:
        - "groups"
        - "email"
      config:
        # ...
    - name: "Validation 2"
      type: "external"
      url: "http://localhost:8080"
      claims:
        - "groups"
      ca: "..."
      config:
        # ...
  mutatingHooks:
    - name: "Mutating 1"
      type: "internal"
      claims:
        - "groups"
      config:
        # ...
    - name: "Mutanting 2"
      type: "external"
      claims:
        - "groups"
      insecureSkipVerify: false
      config:
        # ...
```

Each hook is executed in the order defined in the dex config file.

### Implementation Details/Notes/Constraints

### Risks and Mitigations

#### Internal Hooks

Internal hooks are defined in the same process as Dex. This approach provides better performance, but opens the door to security concerns.
To deal with them, we propose to limit the usage of computation bounded, sandboxed languages as internal hooks, such as [Cel](https://github.com/google/cel-go) or [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/) for internal hooks.

#### External Hooks

External hooks are defined in a separate process and communicate with Dex over gRPC/Rest. This approach would be more flexible but has increased security and performance challenges.

A webhook represents an external dependency that may be unavailable or slow. Therefore, hooks can be configured to be mandatory or optional. In case of failure, the mandatory hooks would lead to a failure of the request, while the optional hooks would be ignored, leading to degraded behavior.

External webhooks have to be trusted by the Dex administrators. The risks of external hooks concern, in particular, the security of abusing the webhook context, which may access some restricted resources and should be accessed only by a legitimate dex instance and the forward of the dex requests to an untrusted webhook.
From a security perspective, webhook endpoints should be protected with TLS and authentication, whose parameters can be configured via the configuration file.
