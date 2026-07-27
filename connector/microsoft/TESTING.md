# Testing the Microsoft Connector

This guide covers how to test the Microsoft connector with both client secret and client assertion authentication methods.

## Prerequisites

- `az` CLI installed and logged in (`az login`)
- `openssl` (for generating keypairs)
- Dex built locally (`make build examples`)
- `jq` (for JSON parsing)
- `kubectl` - for testing using Federated Credential

## Setup Azure AD Application

First, create an Azure AD application that will be used for testing:

```bash
# Set variables
APP_NAME="dex-test-$(date +%s)"
REDIRECT_URI="http://127.0.0.1:5556/dex/callback"

# Create the application
APP_JSON=$(az ad app create \
  --display-name "${APP_NAME}" \
  --sign-in-audience AzureADMyOrg \
  --web-redirect-uris "${REDIRECT_URI}" \
  --query '{appId:appId,displayName:displayName}' \
  -o json)

APP_ID=$(echo "${APP_JSON}" | jq -r .appId)

# Get your tenant ID
TENANT_ID=$(az account show --query tenantId -o tsv)
```

Add Microsoft Graph API permissions:

```bash
# Add User.Read and Directory.Read.All permissions
az ad app permission add \
  --id "${APP_ID}" \
  --api 00000003-0000-0000-c000-000000000000 \
  --api-permissions \
    e1fe6dd8-ba31-4d61-89e7-88639da4683d=Scope \
    06da0dbc-49e2-44d2-8312-53f166ab848a=Scope

# Note: Admin consent may be required for Directory.Read.All
# You can grant it via Azure Portal or with admin permissions:
az ad app permission admin-consent --id "${APP_ID}"
```

## Method 1: Testing with Client Secret

### Create Client Secret

```bash
# Create a client secret (expires in 1 year)
SECRET_JSON=$(az ad app credential reset \
  --id "${APP_ID}" \
  --append \
  --years 1 \
  --query '{secret:password}' \
  -o json)

CLIENT_SECRET=$(echo "${SECRET_JSON}" | jq -r '.secret')
```

### Create Dex Configuration for Client Secret

```bash
# Create temporary config file with client secret
CONFIG_FILE=$(mktemp -t dex-config-secret.yaml)
cat > "${CONFIG_FILE}" <<EOF
issuer: http://127.0.0.1:5556/dex

storage:
  type: sqlite3
  config:
    file: var/dex.db

web:
  http: 127.0.0.1:5556

telemetry:
  http: 127.0.0.1:5558

staticClients:
  - id: example-app
    redirectURIs:
      - 'http://127.0.0.1:5555/callback'
    name: 'Example App'
    secret: ZXhhbXBsZS1hcHAtc2VjcmV0

connectors:
  - type: microsoft
    id: microsoft
    name: Microsoft
    config:
      clientID: "${APP_ID}"
      clientSecret: "${CLIENT_SECRET}"
      tenant: "${TENANT_ID}"
      redirectURI: http://127.0.0.1:5556/dex/callback

enablePasswordDB: true
EOF
```

### Run the Test with Client Secret

```bash
# Terminal 1: Start Dex
./bin/dex serve "${CONFIG_FILE}"

# Terminal 2: Start example app
./bin/example-app --issuer http://127.0.0.1:5556/dex

# Open browser to http://127.0.0.1:5555 and test login with the following:
echo "Test the config with the following:"
echo "Authenticate for: example-app"
echo "Connector ID: microsoft"

# When done, remove the temporary config file
rm "${CONFIG_FILE}"
```

## Method 2: Testing with Client Assertion (JWT)

### Generate RSA Keypair and Certificate

```bash
# Create temporary directory for keys
KEYS_DIR=$(mktemp -d -t microsoft-test-keys.XXXXXX)

# Generate RSA private key (2048-bit)
openssl genrsa -out "${KEYS_DIR}/private.pem" 2048

# Create a self-signed certificate
openssl req -new -x509 \
  -key "${KEYS_DIR}/private.pem" \
  -out "${KEYS_DIR}/cert.pem" \
  -days 365 \
  -subj "/CN=dex-test"
```

### Upload Public Certificate to Azure AD

```bash
# Upload certificate to Azure AD application
az ad app credential reset \
  --id "${APP_ID}" \
  --cert "@${KEYS_DIR}/cert.pem" \
  --append

# Get certificate thumbprint (for JWT header)
THUMBPRINT=$(openssl x509 -in "${KEYS_DIR}/cert.pem" \
  -fingerprint -noout -sha1 | sed 's/://g' | cut -d= -f2)
```

### Create JWT Generation Script

Create the JWT generation script:

```bash
cat > "${KEYS_DIR}/generate-jwt.sh" <<'SCRIPT'
#!/bin/bash
set -e

PRIVATE_KEY="$1"
CERT="$2"
APP_ID="$3"
TENANT_ID="$4"

if [[ -z "${PRIVATE_KEY}" || -z "${CERT}" || -z "${APP_ID}" || -z "${TENANT_ID}" ]]; then
  echo "Usage: $0 <private-key> <cert> <app-id> <tenant-id>"
  exit 1
fi

# Base64URL encode function
base64url() {
  openssl base64 -e -A | tr '+/' '-_' | tr -d '='
}

# Get certificate thumbprint for x5t header
X5T=$(openssl x509 -in "${CERT}" -fingerprint -noout -sha1 | \
  sed 's/://g' | cut -d= -f2 | xxd -r -p | base64url)

# JWT Header
HEADER=$(echo -n "{\"alg\":\"RS256\",\"typ\":\"JWT\",\"x5t\":\"${X5T}\"}" | base64url)

# JWT Claims
NOW=$(date +%s)
EXP=$((NOW + 3600))  # Expires in 1 hour
JTI=$(uuidgen 2>/dev/null || cat /proc/sys/kernel/random/uuid 2>/dev/null || echo "$(date +%s)-$$")

CLAIMS=$(cat <<EOF | tr -d '\n' | tr -d ' '
{
  "aud": "https://login.microsoftonline.com/${TENANT_ID}/oauth2/v2.0/token",
  "exp": ${EXP},
  "iss": "${APP_ID}",
  "jti": "${JTI}",
  "nbf": ${NOW},
  "sub": "${APP_ID}",
  "iat": ${NOW}
}
EOF
)

CLAIMS_B64=$(echo -n "${CLAIMS}" | base64url)

# Create signature
SIGNATURE=$(echo -n "${HEADER}.${CLAIMS_B64}" | \
  openssl dgst -sha256 -sign "${PRIVATE_KEY}" -binary | base64url)

# Output JWT
echo "${HEADER}.${CLAIMS_B64}.${SIGNATURE}"
SCRIPT

chmod +x "${KEYS_DIR}/generate-jwt.sh"
```

### Generate JWT Assertion

```bash
# Generate JWT assertion
"${KEYS_DIR}/generate-jwt.sh" \
  "${KEYS_DIR}/private.pem" \
  "${KEYS_DIR}/cert.pem" \
  "${APP_ID}" \
  "${TENANT_ID}" \
  > "${KEYS_DIR}/assertion.jwt"

```

### Create Dex Configuration for Client Assertion

```bash
# Create temporary config file with client assertion
CONFIG_FILE=$(mktemp -t dex-config-assertion.XXXXXX.yaml)
cat > "${CONFIG_FILE}" <<EOF
issuer: http://127.0.0.1:5556/dex

storage:
  type: sqlite3
  config:
    file: var/dex-assertion.db

web:
  http: 127.0.0.1:5556

telemetry:
  http: 127.0.0.1:5558

staticClients:
  - id: example-app
    redirectURIs:
      - 'http://127.0.0.1:5555/callback'
    name: 'Example App'
    secret: ZXhhbXBsZS1hcHAtc2VjcmV0

connectors:
  - type: microsoft
    id: microsoft-assertion
    name: Microsoft (Client Assertion)
    config:
      clientID: "${APP_ID}"
      clientAssertion: "${KEYS_DIR}/assertion.jwt"
      tenant: "${TENANT_ID}"
      redirectURI: http://127.0.0.1:5556/dex/callback

enablePasswordDB: true
EOF
```

### Run the Test with Client Assertion

```bash
# Regenerate JWT before each test (they expire after 1 hour)
"${KEYS_DIR}/generate-jwt.sh" \
  "${KEYS_DIR}/private.pem" \
  "${KEYS_DIR}/cert.pem" \
  "${APP_ID}" \
  "${TENANT_ID}" \
  > "${KEYS_DIR}/assertion.jwt"

# Terminal 1: Start Dex
./bin/dex serve "${CONFIG_FILE}"

# Terminal 2: Start example app
./bin/example-app --issuer http://127.0.0.1:5556/dex

# Open browser to http://127.0.0.1:5555 and test login
echo "Test the config with the following:"
echo "Authenticate for: example-app"
echo "Connector ID: microsoft-assertion"

# When done, remove the temporary files
rm "${CONFIG_FILE}"
rm -rf "${KEYS_DIR}"
```

## Method 3: Testing with Client Assertion Using Kubernetes Workload Identity

If you have a Kubernetes cluster with a publicly available OIDC issuer configured for service accounts:

### Create Service Account

```bash
kubectl create serviceaccount dex-test -n default
```

### Configure Federated Credential in Azure

```bash
# Get your Kubernetes OIDC issuer
K8S_ISSUER=$(kubectl get --raw /.well-known/openid-configuration | jq -r '.issuer')

# Create federated credential
az ad app federated-credential create \
  --id "${APP_ID}" \
  --parameters "{
    \"name\": \"k8s-dex-test\",
    \"issuer\": \"${K8S_ISSUER}\",
    \"subject\": \"system:serviceaccount:default:dex-test\",
    \"audiences\": [\"api://AzureADTokenExchange\"]
  }"
```

### Generate Kubernetes Token

```bash
# Create temporary file for the Kubernetes token
K8S_TOKEN_FILE=$(mktemp -t k8s-token.jwt)

# Create token with correct audience
kubectl create token dex-test \
  -n default \
  --duration=1h \
  --audience=api://AzureADTokenExchange \
  > "${K8S_TOKEN_FILE}"
```

### Create Dex Configuration for Kubernetes

```bash
# Create temporary config file with Kubernetes token
CONFIG_FILE=$(mktemp -t dex-config-k8s.yaml)
cat > "${CONFIG_FILE}" <<EOF
issuer: http://127.0.0.1:5556/dex

storage:
  type: sqlite3
  config:
    file: var/dex-k8s.db

web:
  http: 127.0.0.1:5556

telemetry:
  http: 127.0.0.1:5558

staticClients:
  - id: example-app
    redirectURIs:
      - 'http://127.0.0.1:5555/callback'
    name: 'Example App'
    secret: ZXhhbXBsZS1hcHAtc2VjcmV0

connectors:
  - type: microsoft
    id: microsoft-k8s
    name: Microsoft (Kubernetes Workload Identity)
    config:
      clientID: "${APP_ID}"
      clientAssertion: "${K8S_TOKEN_FILE}"
      tenant: "${TENANT_ID}"
      redirectURI: http://127.0.0.1:5556/dex/callback

enablePasswordDB: true
EOF
```

### Run Dex

```bash
# Terminal 1: Start Dex
./bin/dex serve "${CONFIG_FILE}"

# Terminal 2: Start example app
./bin/example-app --issuer http://127.0.0.1:5556/dex

# Open browser to http://127.0.0.1:5555 and test login
echo "Test the config with the following:"
echo "Authenticate for: example-app"
echo "Connector ID: microsoft-k8s"

# When done, remove the temporary files
rm "${CONFIG_FILE}"
rm "${K8S_TOKEN_FILE}"
```

## Cleanup

```bash
# Delete the Azure AD application
az ad app delete --id "${APP_ID}"

# Delete Kubernetes resources (if created)
kubectl delete serviceaccount dex-test -n default

# Remove temporary files and directories
rm -f var/dex.db var/dex-assertion.db var/dex-k8s.db
[ -n "${CONFIG_FILE}" ] && rm -f "${CONFIG_FILE}"
[ -n "${KEYS_DIR}" ] && rm -rf "${KEYS_DIR}"
[ -n "${K8S_TOKEN_FILE}" ] && rm -f "${K8S_TOKEN_FILE}"
```

## Notes

- JWTs generated with the script are valid for 1 hour
- Dex reads the assertion file on each authentication request, so you can update it without restarting Dex
- For production use with Kubernetes, the token is automatically rotated by the kubelet
- Client assertions are recommended over client secrets for improved security (no secrets to store)
