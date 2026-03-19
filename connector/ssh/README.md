# SSH Connector

The SSH connector allows users to authenticate using SSH keys instead of passwords. This connector is designed specifically for Kubernetes environments where users want to leverage their existing SSH key infrastructure for authentication.

## Features

- **SSH Key Authentication**: Users authenticate using their SSH keys via SSH agent or key files
- **Dual Authentication Modes**: Supports both JWT-based and challenge/response authentication
- **OAuth2 Token Exchange**: Uses RFC 8693 OAuth2 Token Exchange for standards-compliant authentication
- **Challenge/Response Flow**: Direct SSH signature verification for simpler CLI integration
- **Flexible Key Storage**: Supports both SSH key fingerprints and full public keys in configuration
- **Group Mapping**: Map SSH users to groups for authorization
- **Audit Logging**: Comprehensive authentication event logging
- **Multiple Issuer Support**: Accept JWTs from multiple configured issuers

## Authentication Modes

The SSH connector supports two authentication modes:

### Mode 1: JWT-Based Authentication (OAuth2 Token Exchange)

**Best for**: Sophisticated clients like kubectl-ssh-oidc that need full OAuth2 compliance

1. Client creates a JWT signed with SSH key
2. Client performs OAuth2 Token Exchange using the SSH JWT as subject token
3. Dex validates the JWT via the connector's `TokenIdentity` method
4. Dex returns standard OAuth2 tokens (ID token, access token, refresh token)

### Mode 2: Challenge/Response Authentication (CallbackConnector)

**Best for**: Simple CLI tools and shell scripts that want direct SSH signature verification

1. Client requests authentication URL with `ssh_challenge=true` parameter
2. Dex generates cryptographic challenge and returns it in callback URL
3. Client extracts challenge, signs it with SSH private key
4. Client submits signed challenge to callback URL
5. Dex verifies SSH signature and returns OAuth2 authorization code

**Challenge Expiration**: Challenges expire after the configured `challenge_ttl` (default 300 seconds/5 minutes) and are single-use to prevent replay attacks.

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

    # Input JWT issuer configuration - controls which JWTs Dex will ACCEPT
    # IMPORTANT: These are NOT the same as the issuer of JWTs that Dex produces
    # Dex accepts JWTs with these issuers, but issues its own JWTs with Dex's configured issuer
    allowed_issuers:
      - "kubectl-ssh-oidc"      # Accept JWTs from kubectl-ssh-oidc tool
      - "my-custom-issuer"      # Accept JWTs from custom client tools
      - "ssh-agent-helper"      # Accept JWTs from other SSH authentication tools

    # Dex instance ID for JWT audience validation (SECURITY)
    # This ensures JWTs are created specifically for this Dex instance
    # Should match your Dex issuer URL or a unique instance identifier
    dex_instance_id: "https://dex.example.com"

    # Target audience configuration (for final OIDC tokens)
    # Controls what audiences can be requested in JWT target_audience claim
    # For Kubernetes OIDC, use client IDs as target audiences
    allowed_target_audiences:
      - "kubectl"           # Standard kubectl client ID
      - "example-app"       # Custom application client ID

    # Default groups assigned to all authenticated users
    default_groups: ["authenticated"]

    # Token TTL in seconds (default: 3600)
    token_ttl: 7200

    # Challenge TTL in seconds for challenge/response auth (default: 300)
    challenge_ttl: 600

    # OAuth2 client IDs allowed to use this connector (legacy - use allowed_audiences instead)
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

The SSH connector supports multiple client types:

### JWT-Based Clients

**kubectl-ssh-oidc Plugin**: The [kubectl-ssh-oidc](https://github.com/nikogura/kubectl-ssh-oidc) plugin provides full JWT-based authentication:

```bash
# Install kubectl-ssh-oidc plugin
kubectl ssh-oidc --dex-url https://dex.example.com --client-id kubectl

# The plugin will:
# 1. Generate a JWT signed with your SSH key
# 2. Perform OAuth2 Token Exchange with Dex
# 3. Return Kubernetes credentials
```

### Challenge/Response Clients

**Simple CLI Authentication**: For basic shell scripts and CLI tools:

```bash
#!/bin/bash
# Example CLI client for challenge/response authentication

DEX_URL="https://dex.example.com"
CLIENT_ID="kubectl"
USERNAME="alice"

# Step 1: Request challenge
AUTH_URL=$(curl -s "${DEX_URL}/auth/${CLIENT_ID}/authorize?response_type=code&ssh_challenge=true" \
  | grep -o 'Location: [^"]*' | cut -d' ' -f2)

# Step 2: Extract challenge from auth URL
CHALLENGE=$(echo "$AUTH_URL" | sed -n 's/.*ssh_challenge=\([^&]*\).*/\1/p' | base64 -d)

# Step 3: Sign challenge with SSH key
SIGNATURE=$(echo -n "$CHALLENGE" | ssh-keysign - | base64 -w0)

# Step 4: Submit signed challenge
STATE=$(echo "$AUTH_URL" | sed -n 's/.*state=\([^&]*\).*/\1/p')
CALLBACK_URL=$(echo "$AUTH_URL" | sed -n 's/^\([^?]*\).*/\1/p')

curl -X POST "$CALLBACK_URL" \
  -d "username=$USERNAME" \
  -d "signature=$SIGNATURE" \
  -d "state=$STATE"

# Result: OAuth2 authorization code for token exchange
```

**JWT-Based Clients**: Must use the dual-audience JWT format with both `aud` and `target_audience` claims.

**Challenge/Response Clients**: Use direct SSH signature verification - no JWT required.

## Issuer Configuration: Input vs Output

**CRITICAL DISTINCTION**: The SSH connector configuration deals with **input issuers** (JWTs Dex accepts), which are completely separate from **output issuers** (JWTs Dex produces).

### Input Issuers (`allowed_issuers`)
These control which external JWTs the SSH connector will **accept** for authentication:

```yaml
allowed_issuers:
  - "kubectl-ssh-oidc"      # Accept JWTs from kubectl-ssh-oidc client
  - "ssh-agent-helper"      # Accept JWTs from custom SSH helper tools
  - "my-company-ssh-tool"   # Accept JWTs from internal tools
```

- **Purpose**: Validates the `iss` claim in incoming SSH-signed JWTs
- **Security**: Prevents arbitrary clients from claiming to be trusted issuers
- **Multiple Support**: Can accept JWTs from multiple different client tools
- **Empty List Behavior**: If empty, accepts JWTs from **any** issuer (less secure)

### Output Issuer (Dex Configuration)
This is configured in Dex's main configuration file, **NOT** in the SSH connector:

```yaml
# In dex.yaml (main Dex config)
issuer: https://dex.example.com

connectors:
- type: ssh
  # SSH connector config has NO control over output issuer
```

- **Purpose**: All JWTs that Dex **produces** will have `iss: "https://dex.example.com"`
- **Control**: Completely separate from SSH connector configuration
- **Single Value**: Dex can only have one output issuer URL

### Example Flow
1. **Client creates JWT**: `{"iss": "kubectl-ssh-oidc", "sub": "alice", ...}`
2. **SSH connector validates**: Checks if "kubectl-ssh-oidc" is in `allowed_issuers`
3. **Dex authenticates user**: Verifies SSH signature, creates user session
4. **Dex issues tokens**: `{"iss": "https://dex.example.com", "sub": "alice", ...}`

**Key Point**: The SSH connector accepts JWTs with issuer "kubectl-ssh-oidc" but Dex produces JWTs with issuer "https://dex.example.com". These are completely different values serving different purposes.

## JWT Format and Security Model

**CRITICAL SECURITY NOTICE**: This connector implements a secure JWT verification model where JWT is treated as just a packaging format. The JWT contains NO trusted data until cryptographic verification succeeds.

### JWT Claims Format

The SSH connector expects JWTs with the following standard claims:

**Input JWT (from client to Dex)**:
```json
{
  "sub": "alice",                          // Username (UNTRUSTED until verification)
  "iss": "kubectl-ssh-oidc",              // INPUT issuer - must be in allowed_issuers (UNTRUSTED until verification)
  "aud": "https://dex.example.com",        // Dex instance ID (UNTRUSTED until verification)
  "target_audience": "kubectl",            // Desired token audience (UNTRUSTED until verification)
  "exp": 1234567890,                      // Expiration time (UNTRUSTED until verification)
  "iat": 1234567890,                      // Issued at time (UNTRUSTED until verification)
  "nbf": 1234567890,                      // Not before time (UNTRUSTED until verification)
  "jti": "unique-token-id"                // JWT ID (UNTRUSTED until verification)
}
```

**Output JWT (from Dex to clients)**:
```json
{
  "sub": "alice",                          // Same user, now trusted after SSH verification
  "iss": "https://dex.example.com",        // OUTPUT issuer - from main Dex configuration
  "aud": "kubectl",                        // Final audience (from target_audience above)
  "exp": 1234567890,                      // New expiration time
  "iat": 1234567890,                      // New issued time
  // ... standard OIDC claims
}
```

**Notice**: The `iss` field changes from input ("kubectl-ssh-oidc") to output ("https://dex.example.com"). This is normal and expected.

**Dual Audience Model**
- `aud`: Must match the configured `dex_instance_id` - ensures JWT is for this Dex instance
- `target_audience`: Required claim specifying desired audience for final OIDC tokens

**REQUIRED FORMAT**: All JWTs must use the dual-audience model:
- JWTs **must** include both `aud` and `target_audience` claims

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

### Built-in Security Features

The SSH connector includes several built-in security protections:

**User Enumeration Prevention**:
- **Constant-time responses**: Valid and invalid usernames receive identical response patterns and timing
- **Challenge generation**: All users (valid or invalid) receive challenges to prevent enumeration via timing differences
- **Identical error messages**: Authentication failures use consistent error messages regardless of whether user exists

**Rate Limiting**:
- **IP-based rate limiting**: Maximum 10 authentication attempts per IP address per 5-minute window
- **Automatic cleanup**: Rate limit entries are automatically cleaned up to prevent memory leaks
- **Brute force protection**: Prevents attackers from rapidly trying multiple username/key combinations

**Timing Attack Prevention**:
- **Consistent processing**: Authentication logic takes similar time for valid and invalid users
- **Deferred validation**: Username validation is deferred to prevent timing-based user discovery

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
- **Comprehensive audit logging**: All authentication attempts are logged with structured events including:
  - Authentication attempts (successful and failed)
  - Challenge generation and validation
  - Rate limiting events
  - User enumeration prevention activities
- Monitor SSH connector authentication logs for security events
- Set up alerts for failed authentication attempts and rate limiting triggers
- Regularly review user access and group memberships
- Watch for patterns that may indicate attack attempts

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
**Problem**: The `iss` claim in the INPUT JWT doesn't match any value in `allowed_issuers`

**Solutions**:
- Verify the client's JWT has `iss` claim matching one of the `allowed_issuers` values
- Check client configuration uses correct issuer value (e.g., "kubectl-ssh-oidc")
- Add the client's issuer to the `allowed_issuers` list in SSH connector configuration

**Note**: This error is about INPUT JWTs (client→Dex), not OUTPUT JWTs (Dex→client). The OUTPUT issuer is always Dex's main `issuer` configuration and cannot be changed by the SSH connector.

#### "Too many requests" or Rate Limiting
- **Cause**: IP address has exceeded 10 authentication attempts in 5 minutes
- **Solution**: Wait for the rate limit window to expire (5 minutes)
- **Prevention**: Avoid rapid authentication attempts from the same IP
- **Investigation**: Check audit logs for potential brute force attacks

#### User Enumeration Protection Working
- **Normal behavior**: Both valid and invalid users receive identical responses
- **Expected**: Challenge generation succeeds for all usernames (this is intentional)
- **Security**: Authentication failures happen during signature verification, not user lookup

### Debug Logging
Enable debug logging to troubleshoot authentication issues:

```yaml
logger:
  level: debug
```

This will show detailed authentication flow information and help identify configuration issues.

## Client Requirements

The SSH connector supports two distinct client authentication methods:

### JWT-Based Client Requirements

For clients using JWT-based authentication (OAuth2 Token Exchange):

1. **Required JWT Claims**
   ```json
   {
     "aud": "https://dex.example.com",     // Must match dex_instance_id
     "target_audience": "kubectl"          // Must be in allowed_target_audiences
   }
   ```

2. **Client Configuration**
   Update kubectl-ssh-oidc clients to include:
   ```json
   {
     "dex_instance_id": "https://dex.example.com",
     "target_audience": "kubectl"
   }
   ```

### Challenge/Response Client Requirements

For clients using challenge/response authentication:

1. **No JWT Required** - Uses direct SSH signature verification
2. **Authentication Flow** - Follow the bash example above
3. **SSH Key Access** - Requires access to SSH private key or SSH agent

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