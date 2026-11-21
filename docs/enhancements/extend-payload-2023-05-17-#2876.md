# Dex Enhancement Proposal (DEP) 2876 - 2023-05-17 - Extend payload

## Table of Contents

- [Summary](#summary)
- [Motivation](#motivation)
    - [Goals/Pain](#goals)
    - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [User Experience](#user-experience)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Risks and Mitigations](#risks-and-mitigations)
    - [Alternatives](#alternatives)
- [Future Improvements](#future-improvements)

## Summary

Dex has very rigid claims support. Adding additional claims via a connector requires forking the code and making changes to various internal structures. There are a number of proposals where this discussed, namely #1635 .
 The scope of these is fairly large and we are exploring a possible solution that limits itself to just
mutating of the claims.

## Context

- #1636

## Motivation

### Goals

The working goals of this proposal are:

- Minimal changes to Dex core source
- Provide an optional Go Interface specification connectors can implement to support extending token payload
- Limit the interface to a sing method which allows mutating all claims before Dex core signs and delivers it

### Non-goals

- We will not explore dynamic loading. This is a future topic.

## Proposal

### User Experience

The implementation is fully backwards compatible. Users not requiring this
feature won't have to change anything in their Dex deployment.

We expect that connectors will expose additional configuration options to allow for customizations
of the payload extender functionality. Examples are toggles to include/exclude certain IDP claims

### Implementation Details/Notes/Constraints

We propose adding a new PayloadExtender interface which connectors can choose to implement:

```golang
// PayloadExtender allows connectors to enhance the payload before signing
type PayloadExtender interface {
	ExtendPayload(scopes []string, payload []byte, connectorData []byte) ([]byte, error)
}
```

The `ExtendPayload` method will be called just before the `JWT` is signed by Dex core.

By implementing this interface connectors get a chance to mutate the `id_token` payload
before signature creation. The `scopes` may be used to perform conditional mutation.
The `payload` is passed as a byte array as well as the `connectorData` associated with the authorization request. This allows the connector to pass session specific context via `connectorData`.
The resulting mutated structure is returned including an `error` condition.

> A separate DEP will be created to propopse `Dynamic Scopes support` so payload extending can be application driven.

### Working example

A working example of this interface implementation could look like this:

```golang
func (c *hsdpConnector) ExtendPayload(scopes []string, payload []byte, cdata []byte) ([]byte, error) {
	var cd connectorData
	var originalClaims map[string]interface{}

	c.logger.Info("ExtendPayload called")

	if err := json.Unmarshal(cdata, &cd); err != nil {
		return payload, err
	}
	if err := json.Unmarshal(payload, &originalClaims); err != nil {
		return payload, err
	}

	// Experimental teams
	var teams []string
	teams = append(teams, cd.Introspect.Organizations.ManagingOrganization)
	originalClaims["teams"] = teams

	extendedPayload, err := json.Marshal(originalClaims)
	if err != nil {
		return payload, err
	}
	return extendedPayload, nil
}
```

This results in a `teams` claim being available in the id_token.

### Risks and Mitigations

Since the claims are mutated / extended from the upstream IDP, any consuming application should
be aware of active payload extending of the connector. Care should be taken by the extender not to
break or remove standard claims. All ExtendPayload implementation should include tests to cover
all common scenarios.

Any connector not implementing the `ExtendPayload` interface should function as before.

### Alternatives

- We looked at KeyCloak to support this but Dex is way lighter in deployment and maintenance.
- Commercial IDP offerings have similar filtering/mutating capabilities but have a lock-in disadvantage.

## Future Improvements

- An DEP will be submitted to support `Dynamic scopes support`
- We want to explore implementing a generic pluggable filtering mechanism which is connector agnostic
