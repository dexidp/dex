package sql

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// registerConnectorType discovers and caches sensitive fields for a connector type
// This uses reflection to find struct fields tagged with sensitive:"true"
func (svc *encryptionService) registerConnectorType(connectorType string, configStruct interface{}) {
	if !svc.enabled {
		return
	}

	fields := discoverSensitiveFields(configStruct)
	svc.sensitiveFields[connectorType] = fields

	if len(fields) > 0 {
		svc.logger.Debug("registered connector type for encryption",
			"type", connectorType,
			"sensitive_fields", fields)
	}
}

// encryptFields encrypts all sensitive fields in a connector config JSON blob
func (svc *encryptionService) encryptFields(connectorType string, configJSON []byte) ([]byte, error) {
	if !svc.enabled {
		return configJSON, nil
	}

	fields, ok := svc.sensitiveFields[connectorType]
	if !ok || len(fields) == 0 {
		// No sensitive fields for this connector type
		return configJSON, nil
	}

	// Parse JSON into map
	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to parse connector config JSON: %w", err)
	}

	// Encrypt each sensitive field
	for _, fieldName := range fields {
		if err := svc.encryptFieldInMap(config, fieldName); err != nil {
			return nil, fmt.Errorf("failed to encrypt field %q: %w", fieldName, err)
		}
	}

	// Marshal back to JSON
	encryptedJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal encrypted config: %w", err)
	}

	svc.logger.Debug("encrypted connector fields",
		"type", connectorType,
		"fields", fields)

	return encryptedJSON, nil
}

// decryptFields decrypts all sensitive fields in a connector config JSON blob
func (svc *encryptionService) decryptFields(connectorType string, configJSON []byte) ([]byte, error) {
	if !svc.enabled {
		return configJSON, nil
	}

	fields, ok := svc.sensitiveFields[connectorType]
	if !ok || len(fields) == 0 {
		// No sensitive fields for this connector type
		return configJSON, nil
	}

	// Parse JSON into map
	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to parse connector config JSON: %w", err)
	}

	// Decrypt each sensitive field
	for _, fieldName := range fields {
		if err := svc.decryptFieldInMap(config, fieldName); err != nil {
			// Log warning but continue - support mixed encrypted/unencrypted during migration
			svc.logger.Warn("failed to decrypt field",
				"type", connectorType,
				"field", fieldName,
				"error", err)
		}
	}

	// Marshal back to JSON
	decryptedJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal decrypted config: %w", err)
	}

	svc.logger.Debug("decrypted connector fields",
		"type", connectorType,
		"fields", fields)

	return decryptedJSON, nil
}

// encryptFieldInMap encrypts a single field in a config map
func (svc *encryptionService) encryptFieldInMap(config map[string]interface{}, fieldName string) error {
	value, exists := config[fieldName]
	if !exists {
		// Field not present in config
		return nil
	}

	// Only encrypt string values
	strValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("field %q is not a string (type: %T)", fieldName, value)
	}

	// Skip empty strings
	if strValue == "" {
		return nil
	}

	// Skip if already encrypted
	if isEncrypted(strValue) {
		svc.logger.Debug("field already encrypted, skipping", "field", fieldName)
		return nil
	}

	// Encrypt the value
	encrypted, err := svc.encryptor.encrypt(strValue)
	if err != nil {
		return err
	}

	// Replace with encrypted value
	config[fieldName] = encrypted
	return nil
}

// decryptFieldInMap decrypts a single field in a config map
func (svc *encryptionService) decryptFieldInMap(config map[string]interface{}, fieldName string) error {
	value, exists := config[fieldName]
	if !exists {
		// Field not present in config
		return nil
	}

	// Only decrypt string values
	strValue, ok := value.(string)
	if !ok {
		// Not a string, skip
		return nil
	}

	// Skip if not encrypted (backward compatibility)
	if !isEncrypted(strValue) {
		svc.logger.Debug("field not encrypted, skipping", "field", fieldName)
		return nil
	}

	// Decrypt the value
	decrypted, err := svc.encryptor.decrypt(strValue)
	if err != nil {
		return err
	}

	// Replace with decrypted value
	config[fieldName] = decrypted
	return nil
}

// discoverSensitiveFields uses reflection to find struct fields with sensitive:"true" tag
func discoverSensitiveFields(configStruct interface{}) []string {
	var fields []string

	t := reflect.TypeOf(configStruct)
	if t == nil {
		return fields
	}

	// Dereference pointer if needed
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Only process structs
	if t.Kind() != reflect.Struct {
		return fields
	}

	// Iterate through struct fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Check for sensitive tag
		sensitiveTag := field.Tag.Get("sensitive")
		if sensitiveTag != "true" {
			// TODO: In future, support "required" and "optional" values
			continue
		}

		// Get JSON field name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Extract field name (ignore options like omitempty)
		jsonName := strings.Split(jsonTag, ",")[0]
		fields = append(fields, jsonName)
	}

	return fields
}
