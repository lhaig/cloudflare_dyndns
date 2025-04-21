package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// CloudflareAPIBaseURL is the base URL for Cloudflare API
	CloudflareAPIBaseURL = "https://api.cloudflare.com/client/v4"
)

// CloudflareClient handles interactions with Cloudflare API
type CloudflareClient struct {
	client   *http.Client
	zoneID   string
	apiToken string
}

// NewCloudflareClient creates a new CloudflareClient instance
func NewCloudflareClient(zoneID, apiToken string) *CloudflareClient {
	return &CloudflareClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		zoneID:   zoneID,
		apiToken: apiToken,
	}
}

// GetDNSRecords retrieves DNS records for a hostname and record type
func (c *CloudflareClient) GetDNSRecords(ctx context.Context, recordType, hostname string) ([]DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records?type=%s&name=%s",
		CloudflareAPIBaseURL, c.zoneID, recordType, hostname)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var cfResp CloudflareResponse
	err = json.Unmarshal(body, &cfResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !cfResp.Success {
		return nil, errors.New("cloudflare API request failed")
	}

	return cfResp.Result, nil
}

// CreateDNSRecord creates a new DNS record
func (c *CloudflareClient) CreateDNSRecord(ctx context.Context, recordType, hostname, content string, proxied bool) (bool, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records", CloudflareAPIBaseURL, c.zoneID)

	recordData := DNSRecord{
		Type:    recordType,
		Name:    hostname,
		Content: content,
		Proxied: proxied,
	}

	jsonData, err := json.Marshal(recordData)
	if err != nil {
		return false, fmt.Errorf("marshal record data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("read response: %w", err)
	}

	var cfResp CloudflareUpdateResponse
	err = json.Unmarshal(body, &cfResp)
	if err != nil {
		return false, fmt.Errorf("unmarshal response: %w", err)
	}

	return cfResp.Success, nil
}

// UpdateDNSRecord updates an existing DNS record
func (c *CloudflareClient) UpdateDNSRecord(ctx context.Context, recordID, recordType, hostname, content string, proxied bool) (bool, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", CloudflareAPIBaseURL, c.zoneID, recordID)

	recordData := DNSRecord{
		Type:    recordType,
		Name:    hostname,
		Content: content,
		Proxied: proxied,
	}

	jsonData, err := json.Marshal(recordData)
	if err != nil {
		return false, fmt.Errorf("marshal record data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("read response: %w", err)
	}

	var cfResp CloudflareUpdateResponse
	err = json.Unmarshal(body, &cfResp)
	if err != nil {
		return false, fmt.Errorf("unmarshal response: %w", err)
	}

	return cfResp.Success, nil
}
