package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// DNSUpdater implements Cloudflare DNS updating functionality
type DNSUpdater struct {
	ZoneID     string
	APIToken   string
	Hostname   string
	IPAddr     string
	EnableIPv6 bool
	client     *http.Client
}

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

// NewDNSUpdater creates a new DNS updater instance
func NewDNSUpdater(zoneID, apiToken, hostname, ipAddr string, enableIPv6 bool) *DNSUpdater {
	return &DNSUpdater{
		ZoneID:     zoneID,
		APIToken:   apiToken,
		Hostname:   hostname,
		IPAddr:     ipAddr,
		EnableIPv6: enableIPv6,
		client:     &http.Client{},
	}
}

// Run executes the DNS update process
func (u *DNSUpdater) Run() (string, error) {
	var err error
	var ipv4, ipv6 string

	// If no IP address provided, detect it
	if u.IPAddr == "" {
		ipv4, err = u.detectIPv4()
		if err != nil {
			return "", err
		}
	} else {
		// Check if provided IP is IPv4 or IPv6
		if isIPv4(u.IPAddr) {
			ipv4 = u.IPAddr
		} else if isIPv6(u.IPAddr) {
			ipv6 = u.IPAddr
			u.EnableIPv6 = false // Synology provided IPv6
		} else {
			return "", errors.New("invalid IP address provided")
		}
	}

	// If IPv6 is enabled and we don't have an IPv6 yet, detect it
	if u.EnableIPv6 && ipv6 == "" {
		ipv6, err = u.detectIPv6()
		if err == nil && ipv6 != "" {
			// IPv6 successfully detected
		} else {
			u.EnableIPv6 = false // Disable IPv6 if not available
		}
	}

	// Get current IPv4 record
	var recordID, recordIP string
	var recordProxied bool

	if ipv4 != "" {
		recordType := "A"
		records, err := u.getRecords(recordType)
		if err != nil {
			return "", err
		}

		if len(records) > 0 {
			recordID = records[0].ID
			recordIP = records[0].Content
			recordProxied = records[0].Proxied
		}
	}

	// Get current IPv6 record if enabled
	var recordIDv6, recordIPv6 string
	var recordProxiedv6 bool

	if u.EnableIPv6 && ipv6 != "" {
		recordType := "AAAA"
		records, err := u.getRecords(recordType)
		if err != nil {
			return "", err
		}

		if len(records) > 0 {
			recordIDv6 = records[0].ID
			recordIPv6 = records[0].Content
			recordProxiedv6 = records[0].Proxied
		}
	}

	// Check if IP addresses have changed
	if (ipv4 != "" && recordIP == ipv4) &&
		(ipv6 != "" && recordIPv6 == ipv6 || !u.EnableIPv6) {
		return "no changes needed", nil
	}

	// Update records as needed
	var updateIPv4Success, updateIPv6Success bool

	// Update IPv4 if needed
	if ipv4 != "" && (recordIP != ipv4 || recordID == "") {
		recordType := "A"

		if recordID == "" {
			// Create new record
			success, err := u.createRecord(recordType, ipv4, true)
			if err != nil {
				return "", err
			}
			updateIPv4Success = success
		} else {
			// Update existing record
			success, err := u.updateRecord(recordType, recordID, ipv4, recordProxied)
			if err != nil {
				return "", err
			}
			updateIPv4Success = success
		}
	} else {
		updateIPv4Success = true
	}

	// Update IPv6 if enabled and needed
	if u.EnableIPv6 && ipv6 != "" && (recordIPv6 != ipv6 || recordIDv6 == "") {
		recordType := "AAAA"

		if recordIDv6 == "" {
			// Create new record
			success, err := u.createRecord(recordType, ipv6, true)
			if err != nil {
				return "", err
			}
			updateIPv6Success = success
		} else {
			// Update existing record
			success, err := u.updateRecord(recordType, recordIDv6, ipv6, recordProxiedv6)
			if err != nil {
				return "", err
			}
			updateIPv6Success = success
		}
	} else if !u.EnableIPv6 {
		updateIPv6Success = true
	}

	// Return result
	if updateIPv4Success || updateIPv6Success {
		return "update successful", nil
	}

	return "authentication_failed", errors.New("failed to update DNS records due to authentication error")
}

// getRecords retrieves DNS records from Cloudflare
func (u *DNSUpdater) getRecords(recordType string) ([]DNSRecord, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=%s&name=%s",
		u.ZoneID, recordType, u.Hostname)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+u.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var cfResp CloudflareResponse
	err = json.Unmarshal(body, &cfResp)
	if err != nil {
		return nil, err
	}

	if !cfResp.Success {
		return nil, errors.New("cloudflare API request failed")
	}

	return cfResp.Result, nil
}

// createRecord creates a new DNS record in Cloudflare
func (u *DNSUpdater) createRecord(recordType, content string, proxied bool) (bool, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", u.ZoneID)

	recordData := DNSRecord{
		Type:    recordType,
		Name:    u.Hostname,
		Content: content,
		Proxied: proxied,
	}

	jsonData, err := json.Marshal(recordData)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+u.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var cfResp CloudflareUpdateResponse
	err = json.Unmarshal(body, &cfResp)
	if err != nil {
		return false, err
	}

	return cfResp.Success, nil
}

// updateRecord updates an existing DNS record in Cloudflare
func (u *DNSUpdater) updateRecord(recordType, recordID, content string, proxied bool) (bool, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", u.ZoneID, recordID)

	recordData := DNSRecord{
		Type:    recordType,
		Name:    u.Hostname,
		Content: content,
		Proxied: proxied,
	}

	jsonData, err := json.Marshal(recordData)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+u.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var cfResp CloudflareUpdateResponse
	err = json.Unmarshal(body, &cfResp)
	if err != nil {
		return false, err
	}

	return cfResp.Success, nil
}

// detectIPv4 detects the current public IPv4 address
func (u *DNSUpdater) detectIPv4() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(ip), nil
}

// detectIPv6 detects the current public IPv6 address
func (u *DNSUpdater) detectIPv6() (string, error) {
	// First try to detect IPv6 using external services
	// Try multiple services in case one is down
	ipv6Services := []string{
		"https://api6.ipify.org",
		"https://v6.ident.me/",
		"https://ipv6.icanhazip.com/",
	}

	for _, service := range ipv6Services {
		// Try to get IPv6 address from external service
		client := &http.Client{
			Timeout: 5 * time.Second, // Add timeout for external services
		}
		resp, err := client.Get(service)
		if err == nil {
			defer resp.Body.Close()

			ip, err := io.ReadAll(resp.Body)
			if err == nil {
				ipStr := strings.TrimSpace(string(ip))
				if isIPv6(ipStr) {
					return ipStr, nil
				}
			}
		}
	}

	// Fallback to local network interfaces if external services failed
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", errors.New("failed to get IPv6 address: both external services and local interfaces failed")
	}

	for _, iface := range interfaces {
		// Skip loopback, inactive interfaces, and wireless (which often have temporary IPv6)
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP
			// Skip IPv4 and link-local addresses
			if ip.To4() != nil || ip.IsLinkLocalUnicast() {
				continue
			}

			// Only use global unicast addresses
			if ip.To16() != nil && ip.IsGlobalUnicast() {
				return ip.String(), nil
			}
		}
	}

	return "", errors.New("no IPv6 address found from external services or local interfaces")
}

// isIPv4 checks if a string is an IPv4 address
func isIPv4(ip string) bool {
	regex := `^((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])$`
	r := regexp.MustCompile(regex)
	return r.MatchString(ip)
}

// isIPv6 checks if a string is an IPv6 address
func isIPv6(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && parsedIP.To4() == nil
}
