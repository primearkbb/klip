package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestKeyStore(t *testing.T) (*KeyStore, string) {
	tempDir := t.TempDir()

	// Mock home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() {
		os.Setenv("HOME", oldHome)
	})

	keyStore, err := NewKeyStore()
	if err != nil {
		t.Fatalf("Failed to create KeyStore: %v", err)
	}

	return keyStore, tempDir
}

func TestKeyStore_SaveAndGetKey(t *testing.T) {
	keyStore, _ := setupTestKeyStore(t)

	tests := []struct {
		provider string
		key      string
	}{
		{"anthropic", "test-anthropic-key"},
		{"openai", "test-openai-key"},
		{"openrouter", "test-openrouter-key"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			// Save key
			err := keyStore.SaveKey(tt.provider, tt.key)
			if err != nil {
				t.Fatalf("Failed to save key: %v", err)
			}

			// Get key
			retrievedKey, err := keyStore.GetKey(tt.provider)
			if err != nil {
				t.Fatalf("Failed to get key: %v", err)
			}

			if retrievedKey != tt.key {
				t.Errorf("Expected key %s, got %s", tt.key, retrievedKey)
			}
		})
	}
}

func TestKeyStore_GetKeys(t *testing.T) {
	keyStore, _ := setupTestKeyStore(t)

	// Save multiple keys
	testKeys := map[string]string{
		"anthropic":  "test-anthropic-key",
		"openai":     "test-openai-key",
		"openrouter": "test-openrouter-key",
	}

	for provider, key := range testKeys {
		err := keyStore.SaveKey(provider, key)
		if err != nil {
			t.Fatalf("Failed to save key for %s: %v", provider, err)
		}
	}

	// Get all keys
	keys, err := keyStore.GetKeys()
	if err != nil {
		t.Fatalf("Failed to get keys: %v", err)
	}

	if keys.Anthropic != testKeys["anthropic"] {
		t.Errorf("Expected Anthropic key %s, got %s", testKeys["anthropic"], keys.Anthropic)
	}
	if keys.OpenAI != testKeys["openai"] {
		t.Errorf("Expected OpenAI key %s, got %s", testKeys["openai"], keys.OpenAI)
	}
	if keys.OpenRouter != testKeys["openrouter"] {
		t.Errorf("Expected OpenRouter key %s, got %s", testKeys["openrouter"], keys.OpenRouter)
	}
}

func TestKeyStore_HasKey(t *testing.T) {
	keyStore, _ := setupTestKeyStore(t)

	// Initially no key
	hasKey, err := keyStore.HasKey("anthropic")
	if err != nil {
		t.Fatalf("Failed to check key existence: %v", err)
	}
	if hasKey {
		t.Error("Expected no key initially")
	}

	// Save key
	err = keyStore.SaveKey("anthropic", "test-key")
	if err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Now should have key
	hasKey, err = keyStore.HasKey("anthropic")
	if err != nil {
		t.Fatalf("Failed to check key existence: %v", err)
	}
	if !hasKey {
		t.Error("Expected key to exist")
	}
}

func TestKeyStore_DeleteKey(t *testing.T) {
	keyStore, _ := setupTestKeyStore(t)

	// Save key
	err := keyStore.SaveKey("anthropic", "test-key")
	if err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Delete key
	err = keyStore.DeleteKey("anthropic")
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	// Check key is gone
	hasKey, err := keyStore.HasKey("anthropic")
	if err != nil {
		t.Fatalf("Failed to check key existence: %v", err)
	}
	if hasKey {
		t.Error("Expected key to be deleted")
	}
}

func TestKeyStore_ListKeys(t *testing.T) {
	keyStore, _ := setupTestKeyStore(t)

	// Initially no keys
	providers, err := keyStore.ListKeys()
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("Expected no keys initially, got %d", len(providers))
	}

	// Save some keys
	err = keyStore.SaveKey("anthropic", "test-key-1")
	if err != nil {
		t.Fatalf("Failed to save anthropic key: %v", err)
	}
	err = keyStore.SaveKey("openai", "test-key-2")
	if err != nil {
		t.Fatalf("Failed to save openai key: %v", err)
	}

	// List keys
	providers, err = keyStore.ListKeys()
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}
	if len(providers) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(providers))
	}

	// Check providers are correct
	providerMap := make(map[string]bool)
	for _, provider := range providers {
		providerMap[provider] = true
	}

	if !providerMap["anthropic"] {
		t.Error("Expected anthropic provider in list")
	}
	if !providerMap["openai"] {
		t.Error("Expected openai provider in list")
	}
}

func TestKeyStore_InvalidProvider(t *testing.T) {
	keyStore, _ := setupTestKeyStore(t)

	// Try invalid provider
	err := keyStore.SaveKey("invalid", "test-key")
	if err == nil {
		t.Error("Expected error for invalid provider")
	}

	_, err = keyStore.GetKey("invalid")
	if err == nil {
		t.Error("Expected error for invalid provider")
	}
}

func TestKeyStore_EncryptionPersistence(t *testing.T) {
	keyStore, tempDir := setupTestKeyStore(t)

	// Save a key
	testKey := "test-encryption-key"
	err := keyStore.SaveKey("anthropic", testKey)
	if err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Verify the key file is encrypted (not readable as plain text)
	keyFile := filepath.Join(tempDir, ".klip", "keys.enc")
	data, err := os.ReadFile(keyFile)
	if err != nil {
		t.Fatalf("Failed to read key file: %v", err)
	}

	// The encrypted data should not contain our test key in plain text
	if string(data) == testKey {
		t.Error("Key file appears to contain plain text, not encrypted data")
	}

	// Create a new keystore instance to test persistence
	keyStore2, err := NewKeyStore()
	if err != nil {
		t.Fatalf("Failed to create second KeyStore: %v", err)
	}

	// Should be able to retrieve the same key
	retrievedKey, err := keyStore2.GetKey("anthropic")
	if err != nil {
		t.Fatalf("Failed to get key from second instance: %v", err)
	}

	if retrievedKey != testKey {
		t.Errorf("Expected key %s, got %s", testKey, retrievedKey)
	}
}
