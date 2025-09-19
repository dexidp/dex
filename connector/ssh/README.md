# SSH Connector

The SSH connector allows users to authenticate using SSH keys instead of passwords. This connector is designed specifically for Kubernetes environments where users want to leverage their existing SSH key infrastructure for authentication.

## Features

- **SSH Key Authentication**: Users authenticate using their SSH keys via SSH agent or key files
- **OAuth2 Token Exchange**: Uses RFC 8693 OAuth2 Token Exchange for standards-compliant authentication
- **Flexible Key Storage**: Supports both SSH key fingerprints and full public keys in configuration
- **Group Mapping**: Map SSH users to groups for authorization
- **Audit Logging**: Comprehensive authentication event logging
- **Multiple Issuer Support**: Accept JWTs from multiple configured issuers

## How It Works

The SSH connector uses OAuth2 Token Exchange (RFC 8693):

1. Client creates a JWT signed with SSH key
2. Client performs OAuth2 Token Exchange using the SSH JWT as subject token
3. Dex validates the JWT via the connector's `TokenIdentity` method
4. Dex returns standard OAuth2 tokens (ID token, access token, refresh token)

## Configuration

```yaml
connectors:
- type: ssh
  id: ssh
  name: SSH
  config:
    # User configuration mapping usernames to SSH keys and user info
    users:
      alice:
        keys:
          - "SHA256:abcd1234..."  # SSH key fingerprint
          - "ssh-rsa AAAAB3NzaC1y..."  # Or full public key
        user_info:
          username: "alice"
          email: "alice@example.com"
          groups: ["developers", "admins"]
      bob:
        keys:
          - "SHA256:efgh5678..."
        user_info:
          username: "bob"
          email: "bob@example.com"
          groups: ["developers"]

    # JWT issuer configuration
    allowed_issuers:
      - "kubectl-ssh-oidc"
      - "my-custom-issuer"

    # Default groups assigned to all authenticated users
    default_groups: ["authenticated"]

    # Token TTL in seconds (default: 3600)
    token_ttl: 7200

    # OAuth2 client IDs allowed to use this connector
    allowed_clients:
      - "kubectl"
      - "my-k8s-client"
```

## User Configuration

### SSH Keys
Users can be configured with SSH keys in two formats:

1. **SSH Key Fingerprints**: `SHA256:abcd1234...` (recommended)
2. **Full Public Keys**: `ssh-rsa AAAAB3NzaC1y...` (also supported)

### User Information
Each user must have:
- `username`: The user's login name
- `email`: User's email address (required for Kubernetes OIDC)
- `groups`: Optional list of groups the user belongs to

## Client Integration

The SSH connector is designed to work with the [kubectl-ssh-oidc](https://github.com/nikogura/kubectl-ssh-oidc) plugin, which handles:

- SSH agent interaction
- JWT creation and signing
- OAuth2 flows
- Kubernetes credential management

### Example Usage

```bash
# Install kubectl-ssh-oidc plugin
kubectl ssh-oidc --dex-url https://dex.example.com --client-id kubectl

# The plugin will:
# 1. Generate a JWT signed with your SSH key
# 2. Perform OAuth2 Token Exchange with Dex
# 3. Return Kubernetes credentials
```

## JWT Format and Security Model

**CRITICAL SECURITY NOTICE**: This connector implements a secure JWT verification model where JWT is treated as just a packaging format. The JWT contains NO trusted data until cryptographic verification succeeds.

### JWT Claims Format

The SSH connector expects JWTs with the following standard claims:

```json
{
  "sub": "alice",                          // Username (UNTRUSTED until verification)
  "iss": "kubectl-ssh-oidc",              // Configured issuer (UNTRUSTED until verification)
  "aud": "kubernetes",                     // Audience (UNTRUSTED until verification)
  "exp": 1234567890,                      // Expiration time (UNTRUSTED until verification)
  "iat": 1234567890,                      // Issued at time (UNTRUSTED until verification)
  "nbf": 1234567890,                      // Not before time (UNTRUSTED until verification)
  "jti": "unique-token-id"                // JWT ID (UNTRUSTED until verification)
}
```

**IMPORTANT**: The JWT does NOT contain SSH keys, fingerprints, or any cryptographic material. These would be security vulnerabilities allowing key injection attacks. SSH keys and fingerprints are only used in the Dex administrative configuration, never in JWT tokens sent by clients.

### Security Model: Authentication vs Authorization

This connector maintains strict separation between authentication and authorization:

**Authentication (Cryptographic Proof)**:
- JWT signature is verified against SSH keys configured by administrators in Dex
- Only SSH keys explicitly configured in the `users` section can verify JWTs
- Clients prove they control the private key by successfully signing the JWT
- JWT verification uses a secure 2-pass process following the jwt-ssh-agent-go pattern

**Authorization (Administrative Policy)**:
- User identity, email, groups, and permissions are configured separately by administrators
- No user information comes from the JWT itself - it's all from Dex configuration
- This prevents privilege escalation through client-controlled JWT claims

**Identity Claim and Proof Process**:
1. **Identity Claim**: User sets the `sub` field in the JWT to claim their identity
2. **Cryptographic Proof**: User signs the JWT with their SSH private key to prove they control that identity
3. **Administrative Verification**: Dex verifies the signature against configured SSH keys for that user
4. **Authorization**: Dex returns user attributes (email, groups) from administrative configuration, not JWT claims

### Administrative Control Model

The Dex configuration provides complete control over access:

1. **Connection Authorization**: Only users explicitly configured in the `users` section can authenticate at all
2. **Cryptographic Authentication**: Each user's configured SSH keys define which private keys can "prove" the user's identity
3. **Scope Authorization**: User configuration provides scopes (email, groups) that determine what the authenticated user can access
4. **No Client Control**: Clients cannot influence authorization - they can only cryptographically prove they control a configured private key

### Why This Design Is Secure

1. **No Key Injection**: JWTs cannot contain verification keys that clients control
2. **Administrative Control**: All trusted keys, user mappings, and scopes are configured by Dex administrators
3. **Separation of Concerns**: Authentication (crypto) is separate from authorization (policy)
4. **Standard Compliance**: Uses only standard JWT claims, no custom security-sensitive fields
5. **Allowlist Model**: Only explicitly configured users with specific SSH keys can authenticate

The JWT must be signed using the "SSH" algorithm (custom signing method that integrates with SSH agents).

## Security Considerations

### SSH Key Management
- Use SSH agent for key storage when possible
- Avoid storing unencrypted private keys on disk
- Regularly rotate SSH keys
- Use strong key types (ED25519, RSA 4096-bit)

### Network Security
- Always use HTTPS for Dex endpoints
- Consider network-level restrictions for the `/ssh/token` endpoint
- Implement proper firewall rules

### Audit and Monitoring
- Monitor SSH connector authentication logs
- Set up alerts for failed authentication attempts
- Regularly review user access and group memberships

## Troubleshooting

### Common Issues

#### "JWT parse error: token is unverifiable"
- Verify SSH key is properly configured in users section
- Check that key fingerprint matches the one in the JWT
- Ensure JWT is signed with correct SSH key

#### "User not found or key not authorized"
- Verify username exists in configuration
- Check that SSH key fingerprint matches configured keys
- Confirm user has required SSH key loaded in agent

#### "Invalid issuer"
- Verify issuer claim in JWT matches `allowed_issuers`
- Check client configuration uses correct issuer value

### Debug Logging
Enable debug logging to troubleshoot authentication issues:

```yaml
logger:
  level: debug
```

This will show detailed authentication flow information and help identify configuration issues.

## Status

- **Connector Status**: Alpha (subject to change)
- **Supports Refresh Tokens**: Yes
- **Supports Groups Claim**: Yes
- **Supports Preferred Username Claim**: Yes

## Contributing

The SSH connector is part of a Dex fork and may be contributed back to upstream Dex. When contributing:

1. Ensure all tests pass: `go test ./connector/ssh`
2. Follow Dex coding standards and patterns
3. Update documentation for any configuration changes
4. Add appropriate test coverage for new features