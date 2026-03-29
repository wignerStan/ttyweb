package validate

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Private IP ranges that should be rejected for external API URLs.
var privateCIDRs = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"169.254.0.0/16", // link-local
	"::1/128",        // IPv6 loopback
	"fc00::/7",       // IPv6 unique local
}

// privateNetworks is parsed once at init time.
var privateNetworks []*net.IPNet

func init() {
	privateNetworks = make([]*net.IPNet, 0, len(privateCIDRs))
	for _, cidr := range privateCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("validate: failed to parse private CIDR %q: %v", cidr, err))
		}
		privateNetworks = append(privateNetworks, network)
	}
}

// ErrPrivateIP is returned when an API URL resolves to a private IP.
var ErrPrivateIP = fmt.Errorf("URL resolves to a private or loopback address")

// ErrInvalidScheme is returned when an API URL does not use HTTPS.
var ErrInvalidScheme = fmt.Errorf("URL must use HTTPS scheme")

// localhostNames are hostnames that are explicitly allowed to resolve to
// loopback addresses and use non-HTTPS schemes (for local development).
var localhostNames = map[string]bool{
	"localhost": true,
	"[::1]":     true,
	"127.0.0.1": true,
}

// APIURL validates that a URL is safe for outbound API requests.
// It rejects private/loopback IPs and requires HTTPS for non-localhost URLs.
func APIURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("URL has no host")
	}

	// Allow explicit localhost hostnames without IP checks.
	isLocalhost := localhostNames[host]

	if !isLocalhost {
		// Resolve hostname to IPs to check for private ranges.
		ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
		if err != nil {
			return fmt.Errorf("failed to resolve host %q: %w", host, err)
		}

		if len(ips) == 0 {
			return fmt.Errorf("host %q resolved to no addresses", host)
		}

		for _, ipAddr := range ips {
			ip := ipAddr.IP
			if isPrivateIP(ip) {
				return ErrPrivateIP
			}
		}
	}

	// Require HTTPS unless the host is explicitly localhost.
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && !isLocalhost {
		return ErrInvalidScheme
	}

	return nil
}

// isPrivateIP reports whether the IP falls within a private, loopback,
// or link-local range.
func isPrivateIP(ip net.IP) bool {
	for _, network := range privateNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
