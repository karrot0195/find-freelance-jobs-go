package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Account struct {
	ID       int    `json:"ID"`
	Provider string `json:"Provider"`
	Username string `json:"Username"`
	Enabled  bool   `json:"Enabled"`
}

type Job struct {
	ID          int    `json:"ID"`
	Title       string `json:"Title"`
	URL         string `json:"URL"`
	Category    string `json:"Category"`
	Price       string `json:"Price"`
	Description string `json:"Description"`
	Provider    string `json:"Provider"`
	CreatedAt   string `json:"CreatedAt"`
}

type RestClient struct {
	apiUrl     string
	apiKey     string
	httpClient *http.Client
}

func NewRestClient(apiUrl, apiKey string) *RestClient {
	return &RestClient{
		apiUrl: apiUrl,
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// newRequest builds an http.Request with the API key header attached.
func (r *RestClient) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, r.apiUrl+path, body)
	if err != nil {
		return nil, err
	}
	if r.apiKey != "" {
		req.Header.Set("X-API-Key", r.apiKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (r *RestClient) Ping() error {
	req, err := r.newRequest(http.MethodGet, "/ping", nil)
	if err != nil {
		return err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping failed: %s", resp.Status)
	}
	return nil
}

func (r *RestClient) GetAccounts() []Account {
	req, err := r.newRequest(http.MethodGet, "/GetAccounts", nil)
	if err != nil {
		return nil
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var accounts []Account
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return nil
	}
	return accounts
}

func (r *RestClient) GetJobs() []Job {
	req, err := r.newRequest(http.MethodGet, "/GetJobs", nil)
	if err != nil {
		return nil
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var jobs []Job
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return nil
	}
	return jobs
}

func (r *RestClient) AddAccount(provider, username string) error {
	payload := map[string]string{
		"provider": provider,
		"username": username,
	}
	body, _ := json.Marshal(payload)

	req, err := r.newRequest(http.MethodPost, "/AddAccount", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("add account failed: %s", resp.Status)
	}
	return nil
}

func (r *RestClient) StartCrawlJobs() error {
	req, err := r.newRequest(http.MethodPost, "/StartCrawlJobs", nil)
	if err != nil {
		return err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("crawl failed: %s", resp.Status)
	}
	return nil
}
