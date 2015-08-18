Authorization (Auth-Z) Proposal
===============================

## Components

Core-Auth consisists of various components:

1. A web app for users to authenticate with.
2. A web app for users to manage auth-z policies.
3. An API to serve auth-z queries.
4. A common golang library to: validate auth-n tokens, assert identities from auth-n tokens, and fetch auth-z policies for users.

## Design Strategy

- Users authenticate and are provided a JWT
- API keys are the same format JWT
- Apps use the common library to access core-auth API to fetch auth-z policies etc
- Auth-z policy requests supply an etag, policies are cached with a ttl

## Basic Flow

1. users log in via OAuth and are redirected to app with token (alternatively entities can use pregenerated api tokens)
1. app uses common lib to assert identity from token
1. app uses common lib to fetch auth-z policies (if not cached, or ttl expired) from configured auth-z server
1. app uses common lib to assert permissions to requested resource(s)  
   input: policies, resource CRN(s)  
   output: yes/no, or filtered list of CRNs
1. app responds with: denial, full-results, or filter-results


## Permissions Specification

### Core Resource Namespaces (CRN)

Format: `crn:provider:product:instance:resource-type:resource`

- `provider`: a unique FQDN of the organization that created/maintains the application
- `product`: a product id/name unique to the provider
- `instance`: an individual deployed instance of the product (FQDN, or UUID?)
- `resource-type`: app specific resource type unqiue to the product
- `resource`: the uniqe id of the resource in question (can use `/` for nested resources)

### Resource Namespace Examples

CoreUpdate Examples: 

```
// CoreOS App
crn:coreos.com:coreupdate:public.update.core-os.net:app:e96281a6-d1af-4bde-9a0a-97b76e56dc57
// CoreOS App's "stable" Group
crn:coreos.com:coreupdate:public.update.core-os.net:group:e96281a6-d1af-4bde-9a0a-97b76e56dc57/stable
```

Quay Example:

```
crn:quay.io:enterprise-registry:my-registry.my-company.com:repo:hello-world
```

### Actions

Similar to CRN but defined and registered by individual apps.
Action describes the type of access that should be allowed or denied (for example, read, write, list, delete, startService, and so on)

```
provider:product:name
```

- `provider`: a unique FQDN of the organization that created/maintains the application (same as CRN)
- `product`: a product id/name unique to the provider (same as CRN)
- `name`: the acutal name of the aciton in the product

#### Action Examples

CoreUpdate (general):

```
coreos.com:coreupdate:read
coreos.com:coreupdate:write
coreos.com:coreupdate:delete
```

CoreUpdate (finer grained control):

```
coreos.com:coreupdate:publish
coreos.com:coreupdate:pause
coreos.com:coreupdate:modifyBehavior
coreos.com:coreupdate:modifyChannel
coreos.com:coreupdate:modifyVersion
```

### Policies

```json
{
  "apiVersion": "v1",
  "id": "7CFCE45E-610A-407C-84DB-86A24658B217",
  "label": "my policy",
  "statements": [
    {
      "effect": "allow",
      "resource": ["crn:..."],
      "action": ["crn:..."],
    },
  ]
}
```

Policy:  

- `apiVersion`: the policy API version
- `id`: unique id of the policy
- `label`: human readable label for the policy
- `statements`: the main element for a policy. it is a list of multiple statements defining access.

Statement:  

- `effect`: "allow" or "deny"
- `resource`: the CRN of the resource
- `action`: described in action section

#### Policiy Examples

CoreUpdate Example:

```json
{
  "apiVersion": "v1",
  "id": "rando-uuid",
  "label": "admin",
  "description": "allows full admin access to everything",
  "statements": [
    {
      "effect": "allow",
      "resource": ["crn:coreos.com:coreupdate:public.update.core-os.net:*:*"],
      "action": ["coreos.com:coreupdate:*"],
    },
  ]
}

{
  "apiVersion": "v1",
  "id": "rando-uuid",
  "label": "full-read-only",
  "description": "allows read only access to everything",
  "statements": [
    {
      "effect": "allow",
      "resource": ["crn:coreos.com:coreupdate:public.update.core-os.net:*:*"],
      "action": ["coreos.com:coreupdate:read"],
    },
  ]
}

{
  "apiVersion": "v1",
  "id": "rando-uuid",
  "label": "full-internal-only",
  "description": "allows read access to everything, denys write access to the main CoreOS app",
  "statements": [
    {
      "effect": "allow",
      "resource": ["crn:coreos.com:coreupdate:public.update.core-os.net:*:*"],
      "action": ["coreos.com:coreupdate:read"],
    },
    {
      "effect": "deny",
      "resource": ["crn:coreos.com:coreupdate:public.update.core-os.net:app:e96281a6-d1af-4bde-9a0a-97b76e56dc57"],
      "action": ["coreos.com:coreupdate:write"],
    },
  ]
}
```

### Policy Associations

Policies can be attached to Organizations, Groups, or Users.

### Core Auth API

Should support the following operations:

- create/delete actions
- create/delete resources
- get policies for entity
- TBD
- TBD

# Open Questions

- Do we support resource-based permissions?
- Do we need to include the org in the CRNs?
- Do we version the policies (different than apiVersion) for easy undo/history?
- Should we have an `enabled` flag on policies?
- How to handle public anonymous access?
- Include the AWS equivalent of `NotResource` and `NotAction`?
- Do we need to support conditions in policies? Perhaps we don't need these for v1.
