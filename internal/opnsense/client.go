package opnsense

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Client struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

type HostOverride struct {
	UUID        string `json:"uuid,omitempty"`
	Enabled     string `json:"enabled"`
	Hostname    string `json:"hostname"`
	Domain      string `json:"domain"`
	RR          string `json:"rr"`
	Server      string `json:"server"`
	Description string `json:"description"`
}

type searchResponse struct {
	Rows []HostOverride `json:"rows"`
}

type addResponse struct {
	UUID   string `json:"uuid"`
	Result string `json:"result"`
}

func New(baseURL, apiKey, apiSecret string, tlsSkipVerify bool) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: tlsSkipVerify}, //nolint:gosec
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		httpClient: &http.Client{Transport: tr},
	}
}

func (c *Client) do(method, path string, body interface{}) (*http.Response, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, c.baseURL+path, &buf)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.apiKey, c.apiSecret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, fmt.Errorf("opnsense API %s %s: HTTP %d", method, path, resp.StatusCode)
	}
	return resp, nil
}

func (c *Client) ListHostOverrides() ([]HostOverride, error) {
	resp, err := c.do("GET", "/api/unbound/settings/searchHostOverride?rowCount=1000", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode list response: %w", err)
	}
	return result.Rows, nil
}

func (c *Client) AddHostOverride(h HostOverride) (string, error) {
	resp, err := c.do("POST", "/api/unbound/settings/addHostOverride", map[string]interface{}{"host": h})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result addResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode add response: %w", err)
	}
	if result.Result != "saved" {
		return "", fmt.Errorf("addHostOverride: unexpected result %q", result.Result)
	}
	return result.UUID, nil
}

func (c *Client) DeleteHostOverride(uuid string) error {
	resp, err := c.do("POST", "/api/unbound/settings/delHostOverride/"+uuid, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var result struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode delete response: %w", err)
	}
	if result.Result != "deleted" {
		return fmt.Errorf("delHostOverride: unexpected result %q", result.Result)
	}
	return nil
}

// Reconfigure applies pending Unbound configuration changes.
func (c *Client) Reconfigure() error {
	resp, err := c.do("POST", "/api/unbound/service/reconfigure", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
