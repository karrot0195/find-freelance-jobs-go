package ui

import (
	"sync"

	"job-aggregator/internal/rest"
)

// clientStore holds the active REST client. It is safe to swap at runtime
// (e.g. from the Settings screen) and safe to read from background goroutines.
type clientStore struct {
	mu     sync.RWMutex
	apiUrl string
	apiKey string
	client *rest.RestClient
}

var store = &clientStore{}

func init() {
	// Default endpoint. Can be changed from the Settings screen at runtime.
	store.set("http://localhost:8080", "")
}

func (c *clientStore) set(url, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.apiUrl = url
	c.apiKey = key
	c.client = rest.NewRestClient(url, key)
}

func (c *clientStore) get() *rest.RestClient {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

func (c *clientStore) url() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.apiUrl
}

func (c *clientStore) key() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.apiKey
}

// Convenience wrappers used throughout app.go.
func apiUrl() string           { return store.url() }
func apiKey() string           { return store.key() }
func setApiUrl(u string)       { store.set(u, store.key()) }
func setApiKey(k string)       { store.set(store.url(), k) }
func setApiConfig(u, k string) { store.set(u, k) }
func client() *rest.RestClient { return store.get() }
