package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

// ApiKeys represents the structure for storing multiple provider API keys
type ApiKeys struct {
	Anthropic  string `json:"anthropic,omitempty"`
	OpenAI     string `json:"openai,omitempty"`
	OpenRouter string `json:"openrouter,omitempty"`
}

// KeyStore handles encrypted storage of API keys using AES-GCM encryption
type KeyStore struct {
	configDir string
	keyFile   string
	key       []byte
	logger    *log.Logger
}

// NewKeyStore creates a new KeyStore instance
func NewKeyStore() (*KeyStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".klip")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	keyStore := &KeyStore{
		configDir: configDir,
		keyFile:   filepath.Join(configDir, "keys.enc"),
		logger:    log.New(os.Stderr),
	}

	if err := keyStore.initKey(); err != nil {
		return nil, fmt.Errorf("failed to initialize encryption key: %w", err)
	}

	return keyStore, nil
}

// initKey initializes or loads the encryption key
func (ks *KeyStore) initKey() error {
	keyFile := filepath.Join(ks.configDir, ".key")

	// Try to load existing key
	if data, err := os.ReadFile(keyFile); err == nil {
		key, err := hex.DecodeString(string(data))
		if err != nil {
			return fmt.Errorf("failed to decode existing key: %w", err)
		}
		if len(key) != 32 {
			return fmt.Errorf("invalid key length: expected 32 bytes, got %d", len(key))
		}
		ks.key = key
		return nil
	}

	// Generate new key
	key := make([]byte, 32) // 256-bit key for AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Save key to file with restricted permissions
	keyHex := hex.EncodeToString(key)
	if err := os.WriteFile(keyFile, []byte(keyHex), 0600); err != nil {
		return fmt.Errorf("failed to save encryption key: %w", err)
	}

	ks.key = key
	return nil
}

// encryptData encrypts data using AES-GCM
func (ks *KeyStore) encryptData(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(ks.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptData decrypts data using AES-GCM
func (ks *KeyStore) decryptData(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(ks.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and decrypt
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plaintext, nil
}

// GetKeys retrieves all stored API keys
func (ks *KeyStore) GetKeys() (*ApiKeys, error) {
	// Return empty keys if file doesn't exist
	if _, err := os.Stat(ks.keyFile); os.IsNotExist(err) {
		return &ApiKeys{}, nil
	}

	// Read encrypted data
	encryptedData, err := os.ReadFile(ks.keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted keys: %w", err)
	}

	if len(encryptedData) == 0 {
		return &ApiKeys{}, nil
	}

	// Decrypt data
	jsonData, err := ks.decryptData(encryptedData)
	if err != nil {
		// If decryption fails, the key file might be corrupted or using an old key
		// Log the error and return empty keys instead of failing
		ks.logger.Warn("Failed to decrypt existing keys, starting fresh", "error", err)

		// Backup the corrupted file and start fresh
		backupPath := ks.keyFile + ".corrupted.backup"
		if backupErr := os.Rename(ks.keyFile, backupPath); backupErr != nil {
			ks.logger.Warn("Failed to backup corrupted key file", "error", backupErr)
		} else {
			ks.logger.Info("Backed up corrupted key file", "backup", backupPath)
		}

		return &ApiKeys{}, nil
	}

	// Parse JSON
	var keys ApiKeys
	if err := json.Unmarshal(jsonData, &keys); err != nil {
		// If JSON parsing fails, also start fresh
		ks.logger.Warn("Failed to parse keys JSON, starting fresh", "error", err)
		return &ApiKeys{}, nil
	}

	return &keys, nil
}

// SaveKeys saves all API keys with encryption
func (ks *KeyStore) SaveKeys(keys *ApiKeys) error {
	// Marshal to JSON
	jsonData, err := json.Marshal(keys)
	if err != nil {
		return fmt.Errorf("failed to marshal keys: %w", err)
	}

	// Encrypt data
	encryptedData, err := ks.encryptData(jsonData)
	if err != nil {
		return fmt.Errorf("failed to encrypt keys: %w", err)
	}

	// Write to file with restricted permissions
	if err := os.WriteFile(ks.keyFile, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write encrypted keys: %w", err)
	}

	return nil
}

// GetKey retrieves a specific provider's API key
func (ks *KeyStore) GetKey(provider string) (string, error) {
	keys, err := ks.GetKeys()
	if err != nil {
		return "", err
	}

	switch provider {
	case "anthropic":
		return keys.Anthropic, nil
	case "openai":
		return keys.OpenAI, nil
	case "openrouter":
		return keys.OpenRouter, nil
	default:
		return "", fmt.Errorf("unknown provider: %s", provider)
	}
}

// SaveKey saves a specific provider's API key
func (ks *KeyStore) SaveKey(provider, key string) error {
	keys, err := ks.GetKeys()
	if err != nil {
		return err
	}

	switch provider {
	case "anthropic":
		keys.Anthropic = key
	case "openai":
		keys.OpenAI = key
	case "openrouter":
		keys.OpenRouter = key
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	return ks.SaveKeys(keys)
}

// DeleteKey removes a specific provider's API key
func (ks *KeyStore) DeleteKey(provider string) error {
	keys, err := ks.GetKeys()
	if err != nil {
		return err
	}

	switch provider {
	case "anthropic":
		keys.Anthropic = ""
	case "openai":
		keys.OpenAI = ""
	case "openrouter":
		keys.OpenRouter = ""
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	return ks.SaveKeys(keys)
}

// ListKeys returns a list of providers that have keys stored
func (ks *KeyStore) ListKeys() ([]string, error) {
	keys, err := ks.GetKeys()
	if err != nil {
		return nil, err
	}

	var providers []string
	if keys.Anthropic != "" {
		providers = append(providers, "anthropic")
	}
	if keys.OpenAI != "" {
		providers = append(providers, "openai")
	}
	if keys.OpenRouter != "" {
		providers = append(providers, "openrouter")
	}

	return providers, nil
}

// HasKey checks if a provider has an API key stored
func (ks *KeyStore) HasKey(provider string) (bool, error) {
	key, err := ks.GetKey(provider)
	if err != nil {
		return false, err
	}
	return key != "", nil
}
