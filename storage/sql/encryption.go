package sql

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/dexidp/dex/connector/atlassiancrowd"
	"github.com/dexidp/dex/connector/bitbucketcloud"
	"github.com/dexidp/dex/connector/gitea"
	"github.com/dexidp/dex/connector/github"
	"github.com/dexidp/dex/connector/gitlab"
	"github.com/dexidp/dex/connector/google"
	"github.com/dexidp/dex/connector/keystone"
	"github.com/dexidp/dex/connector/ldap"
	"github.com/dexidp/dex/connector/linkedin"
	"github.com/dexidp/dex/connector/microsoft"
	"github.com/dexidp/dex/connector/oauth"
	"github.com/dexidp/dex/connector/oidc"
	"github.com/dexidp/dex/connector/openshift"
	"github.com/dexidp/dex/connector/saml"
	"github.com/fernet/fernet-go"
)

const encryptedPrefix = "encrypted:"

// encryptionService handles field-level encryption for connector configs stored in SQL
type encryptionService struct {
	encryptor       *fernetEncryptor
	enabled         bool
	logger          *slog.Logger
	sensitiveFields map[string][]string
}

// fernetEncryptor wraps Fernet encryption with support for key rotation
type fernetEncryptor struct {
	primaryKey *fernet.Key
	allKeys    []*fernet.Key
}

// newEncryptionService creates a new encryption service for SQL storage
func newEncryptionService(keys []string, enabled bool, logger *slog.Logger) (*encryptionService, error) {
	svc := &encryptionService{
		enabled:         enabled,
		logger:          logger,
		sensitiveFields: make(map[string][]string),
	}

	if !enabled {
		logger.Info("connector field encryption disabled")
		return svc, nil
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("encryption enabled but no keys provided")
	}

	encryptor, err := newFernetEncryptor(keys)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	svc.encryptor = encryptor
	// Auto-register all connectors when encryption is enabled
	svc.registerConnectors()

	logger.Info("connector field encryption enabled", "key_count", len(keys))
	return svc, nil
}

// newFernetEncryptor creates a Fernet encryptor with support for multiple keys
// First key is used for encryption, all keys are tried for decryption (key rotation)
func newFernetEncryptor(encodedKeys []string) (*fernetEncryptor, error) {
	if len(encodedKeys) == 0 {
		return nil, fmt.Errorf("at least one encryption key required")
	}

	allKeys := make([]*fernet.Key, len(encodedKeys))

	for i, encodedKey := range encodedKeys {
		// Parse as Fernet key (expects base64-encoded 32-byte key)
		key, err := fernet.DecodeKey(encodedKey)
		if err != nil {
			return nil, fmt.Errorf("invalid Fernet key %d: %w", i, err)
		}

		allKeys[i] = key
	}

	return &fernetEncryptor{
		primaryKey: allKeys[0], // First key is primary
		allKeys:    allKeys,
	}, nil
}

// encrypt encrypts plaintext using Fernet and adds encrypted prefix
func (fe *fernetEncryptor) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Encrypt using primary key
	token, err := fernet.EncryptAndSign([]byte(plaintext), fe.primaryKey)
	if err != nil {
		return "", fmt.Errorf("fernet encryption failed: %w", err)
	}

	// Add prefix to mark as encrypted
	return encryptedPrefix + string(token), nil
}

// decrypt decrypts a Fernet token, trying all available keys (supports rotation)
func (fe *fernetEncryptor) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Check for encrypted prefix
	token := strings.TrimPrefix(ciphertext, encryptedPrefix)
	if token == ciphertext {
		// No prefix found - assume plaintext (backward compatibility during migration)
		return ciphertext, nil
	}

	// Try to decrypt with all keys (supports key rotation)
	plaintext := fernet.VerifyAndDecrypt([]byte(token), 0, fe.allKeys)
	if plaintext == nil {
		return "", fmt.Errorf("fernet decryption failed: invalid token or wrong key")
	}

	return string(plaintext), nil
}

// isEncrypted checks if a value has the encrypted prefix
func isEncrypted(value string) bool {
	return strings.HasPrefix(value, encryptedPrefix)
}

// IsEnabled returns whether encryption is enabled
func (svc *encryptionService) IsEnabled() bool {
	return svc.enabled
}

// hasEncryptedFields checks if a connector config JSON contains any encrypted field values
func (svc *encryptionService) hasEncryptedFields(configJSON []byte) bool {
	// Parse JSON config
	var config map[string]any
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return false
	}

	// Check if any field value has the encrypted prefix
	for _, value := range config {
		if strValue, ok := value.(string); ok && isEncrypted(strValue) {
			return true
		}
	}

	return false
}

func (svc *encryptionService) registerConnectors() {
	svc.registerConnectorType("atlassian-crowd", atlassiancrowd.Config{})
	svc.registerConnectorType("bitbucket-cloud", bitbucketcloud.Config{})
	svc.registerConnectorType("gitea", gitea.Config{})
	svc.registerConnectorType("github", github.Config{})
	svc.registerConnectorType("gitlab", gitlab.Config{})
	svc.registerConnectorType("google", google.Config{})
	svc.registerConnectorType("keystone", keystone.Config{})
	svc.registerConnectorType("ldap", ldap.Config{})
	svc.registerConnectorType("linkedin", linkedin.Config{})
	svc.registerConnectorType("microsoft", microsoft.Config{})
	svc.registerConnectorType("oauth", oauth.Config{})
	svc.registerConnectorType("oidc", oidc.Config{})
	svc.registerConnectorType("openshift", openshift.Config{})
	svc.registerConnectorType("saml", saml.Config{})
}
