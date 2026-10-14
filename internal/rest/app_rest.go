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
	httpClient *http.Client
}

func NewRestClient(apiUrl string) *RestClient {
	return &RestClient{
		apiUrl: apiUrl,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (r *RestClient) Ping() error {
	resp, err := r.httpClient.Get(r.apiUrl + "/ping")
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
	resp, err := r.httpClient.Get(r.apiUrl + "/GetAccounts")
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
	resp, err := r.httpClient.Get(r.apiUrl + "/GetJobs")
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

	resp, err := r.httpClient.Post(
		r.apiUrl+"/AddAccount",
		"application/json",
		bytes.NewReader(body),
	)
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
	resp, err := r.httpClient.Post(r.apiUrl+"/StartCrawlJobs", "application/json", nil)
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
