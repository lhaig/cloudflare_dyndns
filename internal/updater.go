package internal

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Response status constants
const (
	StatusNoChange           = "no changes needed"
	StatusGood               = "update successful"
	StatusAuthError          = "authentication error"
	StatusMissingCredentials = "missing credentials"

	// Record types
	RecordTypeA    = "A"
	RecordTypeAAAA = "AAAA"

	// Default timeout for operations
	defaultTimeout = 30 * time.Second
)

// DNSRecord represents a Cloudflare DNS record
type DNSRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
}

// CloudflareResponse represents response from Cloudflare API
type CloudflareResponse struct {
	Success bool        `json:"success"`
	Result  []DNSRecord `json:"result"`
}

// CloudflareUpdateResponse represents response from Cloudflare update API
type CloudflareUpdateResponse struct {
	Success bool      `json:"success"`
	Result  DNSRecord `json:"result"`
}

// DNSUpdater implements Cloudflare DNS updating functionality
type DNSUpdater struct {
	ZoneID     string
	APIToken   string
	Hostname   string
	IPAddr     string
	EnableIPv6 bool
	cfClient   *CloudflareClient
	ipDetector *IPDetector
}

// NewDNSUpdater creates a new DNS updater instance
func NewDNSUpdater(zoneID, apiToken, hostname, ipAddr string, enableIPv6 bool) *DNSUpdater {
	return &DNSUpdater{
		ZoneID:     zoneID,
		APIToken:   apiToken,
		Hostname:   hostname,
		IPAddr:     ipAddr,
		EnableIPv6: enableIPv6,
		cfClient:   NewCloudflareClient(zoneID, apiToken),
		ipDetector: NewIPDetector(),
	}
}

// Run executes the DNS update process
func (u *DNSUpdater) Run() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Step 1: Detect or validate IP addresses
	ipv4, ipv6, err := u.detectIPs(ctx)
	if err != nil {
		return StatusAuthError, fmt.Errorf("IP detection failed: %w", err)
	}

	// Step 2: Get existing DNS records
	v4Record, err := u.getCurrentRecord(ctx, RecordTypeA)
	if err != nil {
		return StatusAuthError, err
	}

	var v6Record *DNSRecordInfo
	if u.EnableIPv6 && ipv6 != "" {
		v6Record, err = u.getCurrentRecord(ctx, RecordTypeAAAA)
		if err != nil {
			return StatusAuthError, err
		}
	}

	// Step 3: Check if IP addresses have changed
	if !u.needsUpdate(ipv4, ipv6, v4Record, v6Record) {
		return StatusNoChange, nil
	}

	// Step 4: Update records as needed
	updateResult, err := u.updateRecords(ctx, ipv4, ipv6, v4Record, v6Record)
	if err != nil {
		return StatusAuthError, err
	}

	return updateResult, nil
}

// detectIPs detects or validates IP addresses
func (u *DNSUpdater) detectIPs(ctx context.Context) (ipv4, ipv6 string, err error) {
	// If no IP address provided, detect it
	if u.IPAddr == "" {
		ipv4, err = u.ipDetector.DetectIPv4(ctx)
		if err != nil {
			return "", "", fmt.Errorf("detect IPv4: %w", err)
		}
	} else {
		// Check if provided IP is IPv4 or IPv6
		if IsIPv4(u.IPAddr) {
			ipv4 = u.IPAddr
		} else if IsIPv6(u.IPAddr) {
			ipv6 = u.IPAddr
			u.EnableIPv6 = false // User provided IPv6 directly
		} else {
			return "", "", errors.New("invalid IP address provided")
		}
	}

	// If IPv6 is enabled and we don't have an IPv6 yet, detect it
	if u.EnableIPv6 && ipv6 == "" {
		ipv6, err = u.ipDetector.DetectIPv6(ctx)
		if err != nil {
			// If we can't detect IPv6, just disable it rather than failing
			u.EnableIPv6 = false
		}
	}

	return ipv4, ipv6, nil
}

// DNSRecordInfo contains information about a DNS record
type DNSRecordInfo struct {
	ID      string
	Content string
	Proxied bool
}

// getCurrentRecord retrieves the current DNS record of specified type
func (u *DNSUpdater) getCurrentRecord(ctx context.Context, recordType string) (*DNSRecordInfo, error) {
	records, err := u.cfClient.GetDNSRecords(ctx, recordType, u.Hostname)
	if err != nil {
		return nil, fmt.Errorf("get %s records: %w", recordType, err)
	}

	if len(records) == 0 {
		return nil, nil // No record exists
	}

	return &DNSRecordInfo{
		ID:      records[0].ID,
		Content: records[0].Content,
		Proxied: records[0].Proxied,
	}, nil
}

// needsUpdate determines if DNS records need to be updated
func (u *DNSUpdater) needsUpdate(ipv4, ipv6 string, v4Record, v6Record *DNSRecordInfo) bool {
	// For IPv4
	if ipv4 != "" && (v4Record == nil || v4Record.Content != ipv4) {
		return true
	}

	// For IPv6
	if u.EnableIPv6 && ipv6 != "" && (v6Record == nil || v6Record.Content != ipv6) {
		return true
	}

	return false
}

// updateRecords updates the DNS records as needed
func (u *DNSUpdater) updateRecords(ctx context.Context, ipv4, ipv6 string, v4Record, v6Record *DNSRecordInfo) (string, error) {
	var v4Success, v6Success bool
	var v4Err, v6Err error

	// Update IPv4 record if needed
	if ipv4 != "" && (v4Record == nil || v4Record.Content != ipv4) {
		v4Success, v4Err = u.updateRecord(ctx, RecordTypeA, v4Record, ipv4)
	} else {
		v4Success = true
	}

	// Update IPv6 record if enabled and needed
	if u.EnableIPv6 && ipv6 != "" && (v6Record == nil || v6Record.Content != ipv6) {
		v6Success, v6Err = u.updateRecord(ctx, RecordTypeAAAA, v6Record, ipv6)
	} else if !u.EnableIPv6 {
		v6Success = true
	}

	// Handle errors
	if !v4Success && v4Err != nil {
		return StatusAuthError, fmt.Errorf("update IPv4 record: %w", v4Err)
	}

	if !v6Success && v6Err != nil {
		return StatusAuthError, fmt.Errorf("update IPv6 record: %w", v6Err)
	}

	if v4Success || v6Success {
		return StatusGood, nil
	}

	return StatusAuthError, errors.New("failed to update DNS records")
}

// updateRecord updates or creates a single DNS record
func (u *DNSUpdater) updateRecord(ctx context.Context, recordType string, record *DNSRecordInfo, ipContent string) (bool, error) {
	if record == nil {
		// Create new record
		return u.cfClient.CreateDNSRecord(ctx, recordType, u.Hostname, ipContent, true)
	}
	// Update existing record
	return u.cfClient.UpdateDNSRecord(ctx, record.ID, recordType, u.Hostname, ipContent, record.Proxied)
}
