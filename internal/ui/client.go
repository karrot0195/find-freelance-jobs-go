package ui

import (
	"sync"

	"job-aggregator/internal/rest"
)

const (
	keyAPIURL = "api_url"
	keyAPIKey = "api_key"

	defaultAPIURL = "http://localhost:8080"
)

type clientStore struct {
	mu     sync.RWMutex
	apiUrl string
	apiKey string
	client *rest.RestClient
	store  *Store
}

var store = &clientStore{}

// InitStore wires the persistent backend into the client store and
// loads any previously saved configuration. Call once at startup.
func InitStore(s *Store) {
	store.mu.Lock()
	store.store = s
	store.mu.Unlock()

	url, err := s.Get(keyAPIURL)
	if err != nil {
		url = defaultAPIURL
	}
	key, err := s.Get(keyAPIKey)
	if err != nil {
		key = ""
	}
	store.set(url, key)
}

func (c *clientStore) set(url, key string) {
	c.mu.Lock()
	c.apiUrl = url
	c.apiKey = key
	c.client = rest.NewRestClient(url, key)
	s := c.store
	c.mu.Unlock()

	if s != nil {
		// Best-effort persistence; log if you want, but don't block the UI.
		_ = s.Save(keyAPIURL, url)
		_ = s.Save(keyAPIKey, key)
	}
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
