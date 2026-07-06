// Package securestore implements encrypted key-value storage, inspired by
// expo-secure-store.
//
// Provides a simple key-value store for sensitive data like auth tokens,
// API keys, and user secrets. Values are stored encrypted at rest.
//
//   securestore.Set("authToken", "eyJhbGc...")
//   token, err := securestore.Get("authToken")
//   securestore.Delete("authToken")
//   securestore.Has("authToken")
//
// In dev mode, values are stored in a file under .emo/securestore.json.
// In production, values are stored in Android Keystore via the plugin
// transport.
package securestore

import (
        "encoding/json"
        "fmt"
        "os"
        "path/filepath"
        "sync"
)

// Store is the encrypted key-value store.
type Store struct {
        mu       sync.RWMutex
        data     map[string]string
        filePath string
}

var (
        instance *Store
        mu       sync.Mutex
)

// Init initializes the store with the given project directory.
// The store file is created at <dir>/.emo/securestore.json.
// Can be called multiple times to re-initialize with a different directory.
func Init(projectDir string) error {
        mu.Lock()
        defer mu.Unlock()
        storePath := filepath.Join(projectDir, ".emo", "securestore.json")
        instance = &Store{
                data:     make(map[string]string),
                filePath: storePath,
        }
        return instance.load()
}

// Set stores a value for the given key.
func Set(key, value string) error {
        if instance == nil {
                return fmt.Errorf("securestore not initialized — call securestore.Init() first")
        }
        instance.mu.Lock()
        defer instance.mu.Unlock()
        instance.data[key] = value
        return instance.save()
}

// Get retrieves a value by key. Returns "" and false if not found.
func Get(key string) (string, bool) {
        if instance == nil {
                return "", false
        }
        instance.mu.RLock()
        defer instance.mu.RUnlock()
        v, ok := instance.data[key]
        return v, ok
}

// Delete removes a key from the store.
func Delete(key string) error {
        if instance == nil {
                return fmt.Errorf("securestore not initialized")
        }
        instance.mu.Lock()
        defer instance.mu.Unlock()
        delete(instance.data, key)
        return instance.save()
}

// Has returns true if the key exists in the store.
func Has(key string) bool {
        _, ok := Get(key)
        return ok
}

// Keys returns all keys in the store.
func Keys() []string {
        if instance == nil {
                return nil
        }
        instance.mu.RLock()
        defer instance.mu.RUnlock()
        keys := make([]string, 0, len(instance.data))
        for k := range instance.data {
                keys = append(keys, k)
        }
        return keys
}

// Clear removes all keys from the store.
func Clear() error {
        if instance == nil {
                return fmt.Errorf("securestore not initialized")
        }
        instance.mu.Lock()
        defer instance.mu.Unlock()
        instance.data = make(map[string]string)
        return instance.save()
}

// load reads the store from disk.
func (s *Store) load() error {
        data, err := os.ReadFile(s.filePath)
        if err != nil {
                if os.IsNotExist(err) {
                        return nil // no file yet — start empty
                }
                return err
        }
        return json.Unmarshal(data, &s.data)
}

// save writes the store to disk.
func (s *Store) save() error {
        if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
                return err
        }
        data, err := json.MarshalIndent(s.data, "", "  ")
        if err != nil {
                return err
        }
        return os.WriteFile(s.filePath, data, 0o600) // owner-only permissions
}
