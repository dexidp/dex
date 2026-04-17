# Authentication through Google Cloud Identity-Aware Proxy (IAP)

## Overview

The `gcloud-iap` connector validates requests that have been pre-authenticated by
[Google Cloud Identity-Aware Proxy (IAP)][iap-docs]. IAP signs every proxied
request with an ES256 JWT in the `X-Goog-IAP-JWT-Assertion` HTTP header. This
connector verifies that signature cryptographically, extracts the user's identity
from the JWT claims, and optionally enriches the identity with Google Workspace
group membership via the Admin Directory API — **without** domain-wide delegation.

This connector is the right choice when:

- Dex sits behind a Google Cloud IAP-protected load balancer.
- You want cryptographic verification of the IAP assertion (vs. trusting raw
  headers as the `authproxy` connector does).
- You want group membership resolved from Google Workspace using workload
  identity (no super-admin impersonation / DWD required).

## Comparison with existing connectors

| Feature | `authproxy` | `oidc` | `gcloud-iap` |
|---|---|---|---|
| Verifies IAP JWT signature | ✗ | ✗ | ✓ |
| Workspace group membership | ✗ | ✗ | ✓ |
| Requires domain-wide delegation | — | — | ✗ |
| Works behind non-Google proxies | ✓ | — | ✗ |

## Configuration

### Minimal (no group resolution)

```yaml
connectors:
  - type: gcloud-iap
    id: gcloud-iap
    name: Google IAP
    config:
      # Required. IAP backend-service audience.
      # Format: /projects/<project-number>/global/backendServices/<service-id>
      # Find it: gcloud compute backend-services describe <name> --global
      audience: /projects/123456789012/global/backendServices/1234567890123456789
```

No Admin Directory API call is made. The `groups` field in the returned identity
will always be empty.

### With group membership — all groups (single domain)

```yaml
connectors:
  - type: gcloud-iap
    id: gcloud-iap
    name: Google IAP
    config:
      audience: /projects/123456789012/global/backendServices/1234567890123456789

      # Scope the group lookup to one Workspace domain.
      domain: example.com

      # "*" fetches all groups the user belongs to and surfaces them in the
      # identity without any restriction. Every user can log in.
      groupsFilter:
        - "*"
```

### With group membership — restrict to specific patterns

```yaml
connectors:
  - type: gcloud-iap
    id: gcloud-iap
    name: Google IAP
    config:
      audience: /projects/123456789012/global/backendServices/1234567890123456789

      domain: example.com

      # Only users who belong to at least one matching group can log in.
      groupsFilter:
        - "platform-*@example.com"
        - "sre@example.com"

      # Optional: also resolve transitive (nested) group membership.
      # Default: false
      fetchTransitiveGroupMembership: true
```

### With group membership — multi-domain organisation (customerID)

```yaml
connectors:
  - type: gcloud-iap
    id: gcloud-iap
    name: Google IAP
    config:
      audience: /projects/123456789012/global/backendServices/1234567890123456789

      # Use customerID instead of domain for multi-domain Workspace accounts.
      # Find your customer ID in the Admin console under Account → Account settings.
      customerID: C01abc123

      groupsFilter:
        - "*@example.com"
        - "*@subsidiary.com"
```

### All fields

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `audience` | string | **yes** | — | IAP backend-service audience string. Format: `/projects/<number>/global/backendServices/<id>` |
| `iapIssuer` | string | no | `https://cloud.google.com/iap` | Expected `iss` claim in the IAP JWT. Only change this for testing. |
| `iapJWKSUrl` | string | no | `https://www.gstatic.com/iap/verify/public_key-jwk` | JWKS URL used to fetch IAP's public signing keys. Only change this for testing. |
| `domain` | string | see note | — | Scopes group lookup to a single Workspace domain. Mutually exclusive with `customerID`. Required when `groupsFilter` is set and `customerID` is not. |
| `customerID` | string | see note | — | Scopes group lookup to all domains in a Workspace customer account (e.g. `C01abc123`). Mutually exclusive with `domain`. Required when `groupsFilter` is set and `domain` is not. |
| `groupsFilter` | list of strings | no | `[]` | Glob patterns controlling group resolution. See **Pattern syntax** below. When empty, no API call is made. |
| `fetchTransitiveGroupMembership` | bool | no | `false` | Resolve nested group membership recursively. |

### Pattern syntax

Patterns follow standard shell glob rules. Because group email addresses never
contain `/`, the `*` wildcard effectively matches any substring.
Matching is **case-insensitive**.

| Pattern | Meaning |
|---|---|
| `*` | All groups — fetches everything, no login restriction |
| `*@example.com` | All groups in the `example.com` domain |
| `platform-*@example.com` | Groups whose local part starts with `platform-` |
| `sre@example.com` | Exact match (equivalent to a plain string) |

Multiple patterns are **OR-ed**: a user passes as long as at least one of their
groups matches at least one pattern. If `groupsFilter` is non-empty and none of
the user's groups match, **login is denied**.

> **`domain` vs `customerID`**
>
> - Use **`domain`** for single-domain Workspace organisations, or when you want
>   to scope lookups to one domain only.
> - Use **`customerID`** for multi-domain organisations. The value is the numeric
>   Workspace customer ID visible in the Admin console under
>   **Account → Account settings**.
> - The `my_customer` alias is **not** supported. Because the connector
>   authenticates as a workload-identity service account (not as a Workspace
>   member), `my_customer` does not resolve correctly. Use your numeric customer
>   ID instead.

## Finding the audience string

```bash
# Get the numeric project number
gcloud projects describe <project-id> --format='value(projectNumber)'

# Get the numeric backend service ID
gcloud compute backend-services describe <backend-service-name> \
  --global \
  --format='value(id)'
```

The audience string has the form:
`/projects/<project-number>/global/backendServices/<service-id>`

## Group membership — workload identity setup

The connector calls the [Admin Directory API][admin-api] as the workload identity
service account. No domain-wide delegation is required — the SA authenticates
**as itself** and the API call uses `groups.list` with a `userKey` filter.

### Prerequisites

1. **Assign the Groups Reader admin role to the service account.**

   - Open the [Google Workspace Admin console][admin-console].
   - Navigate to **Account → Admin roles**.
   - Select the **Groups Reader** role (a built-in read-only role that grants
     access to the Admin Directory API `groups.list` endpoint).
   - Click **Assign service accounts** and add the email address of the GCP
     service account used by workload identity (e.g.
     `dex-sa@my-project.iam.gserviceaccount.com`).

   This gives the service account read-only access to group membership without
   domain-wide delegation or OAuth scope grants.

2. **Ensure Dex runs with the workload identity SA attached.**
   On GKE this means annotating the Kubernetes service account used by the Dex
   pod with `iam.gke.io/gcp-service-account=<sa-email>` and binding
   `roles/iam.workloadIdentityUser` on the GCP SA.

   No `serviceAccountFilePath` is needed — the connector picks up the workload
   identity credential automatically via Application Default Credentials.

## Callback routing

Like the `authproxy` connector, `gcloud-iap` uses dex's existing
`/callback/{connector}` route. No additional ingress rules are needed beyond
what is already required to reach dex's callback endpoint.

When IAP intercepts the initial `/auth/{connector}` redirect, it authenticates
the user and then forwards the request (with the `X-Goog-IAP-JWT-Assertion`
header set) to dex's `/callback/gcloud-iap` endpoint where the JWT is verified.

[iap-docs]: https://cloud.google.com/iap/docs/concepts-overview
[admin-api]: https://developers.google.com/admin-sdk/directory/reference/rest/v1/groups/list
[admin-console]: https://admin.google.com

