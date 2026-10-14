package scraper

import "fmt"

type JobItem struct {
	Title       string
	URL         string
	Category    string
	Price       string
	Description string // NEW
}

// JobProvider defines the standard contract every job site must implement
type JobProvider interface {
	Name() string
	FetchJobs(username string) ([]JobItem, error)
}

// Registry stores all registered job providers
var providers = make(map[string]JobProvider)

func Register(p JobProvider) {
	providers[p.Name()] = p
}

func FetchJobs(providerName, username string) ([]JobItem, error) {
	provider, exists := providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}
	return provider.FetchJobs(username)
}

func GetSupportedProviders() []string {
	var list []string
	for name := range providers {
		list = append(list, name)
	}
	return list
}
