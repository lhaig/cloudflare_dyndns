package internal

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// IPDetector provides functionality to detect public IP addresses
type IPDetector struct {
	client *http.Client
}

// NewIPDetector creates a new IPDetector
func NewIPDetector() *IPDetector {
	return &IPDetector{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// DetectIPv4 detects the current public IPv4 address
func (d *IPDetector) DetectIPv4(ctx context.Context) (string, error) {
	// Try multiple services for redundancy
	ipv4Services := []string{
		"https://api.ipify.org",
		"https://v4.ident.me/",
		"https://ipv4.icanhazip.com/",
	}

	for _, service := range ipv4Services {
		req, err := http.NewRequestWithContext(ctx, "GET", service, nil)
		if err != nil {
			continue
		}

		resp, err := d.client.Do(req)
		if err != nil {
			continue
		}

		defer resp.Body.Close()
		ip, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		ipStr := strings.TrimSpace(string(ip))
		if IsIPv4(ipStr) {
			return ipStr, nil
		}
	}

	return "", errors.New("failed to detect public IPv4 address")
}

// DetectIPv6 detects the current public IPv6 address
func (d *IPDetector) DetectIPv6(ctx context.Context) (string, error) {
	// First try to detect IPv6 using external services
	ipv6Services := []string{
		"https://api6.ipify.org",
		"https://v6.ident.me/",
		"https://ipv6.icanhazip.com/",
	}

	for _, service := range ipv6Services {
		req, err := http.NewRequestWithContext(ctx, "GET", service, nil)
		if err != nil {
			continue
		}

		resp, err := d.client.Do(req)
		if err != nil {
			continue
		}

		defer resp.Body.Close()
		ip, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		ipStr := strings.TrimSpace(string(ip))
		if IsIPv6(ipStr) {
			return ipStr, nil
		}
	}

	// Fallback to local network interfaces if external services failed
	ipv6, err := detectIPv6FromInterfaces()
	if err == nil {
		return ipv6, nil
	}

	return "", errors.New("no IPv6 address found from external services or local interfaces")
}

// detectIPv6FromInterfaces tries to find a public IPv6 address from network interfaces
func detectIPv6FromInterfaces() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Skip loopback, inactive interfaces
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

	return "", errors.New("no suitable IPv6 address found on interfaces")
}

// IsIPv4 checks if a string is an IPv4 address
func IsIPv4(ip string) bool {
	regex := `^((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])$`
	r := regexp.MustCompile(regex)
	return r.MatchString(ip)
}

// IsIPv6 checks if a string is an IPv6 address
func IsIPv6(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && parsedIP.To4() == nil
}
