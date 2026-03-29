package validate

import (
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip      string
		private bool
	}{
		// Private IPv4 ranges
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"172.15.255.255", false}, // just below 172.16
		{"172.32.0.0", false},     // just above 172.31
		{"192.168.0.1", true},
		{"192.168.255.255", true},
		{"192.167.255.255", false}, // just below 192.168
		{"192.169.0.0", false},     // just above 192.168

		// Loopback
		{"127.0.0.1", true},
		{"127.255.255.255", true},

		// Link-local
		{"169.254.0.1", true},
		{"169.254.255.255", true},
		{"169.253.255.255", false}, // just below
		{"169.255.0.0", false},     // just above

		// Public IPs
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"203.0.113.1", false}, // TEST-NET-3, but not in our private list

		// IPv6 loopback
		{"::1", true},

		// IPv6 unique local
		{"fc00::1", true},
		{"fd00::1", true},
		{"fe00::1", false}, // just below fc00::/7 range in IPv6 space

		// IPv6 public
		{"2001:4860:4860::8888", false}, // Google DNS
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tt.ip)
			}
			got := isPrivateIP(ip)
			if got != tt.private {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tt.ip, got, tt.private)
			}
		})
	}
}

func TestAPIURL_RejectsPrivateIPs(t *testing.T) {
	// These URLs resolve to private IPs or use private ranges directly.
	// We test with raw IP addresses to avoid DNS resolution issues in CI.
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "private 10.x IP with HTTPS",
			url:     "https://10.0.0.1/v1",
			wantErr: true,
		},
		{
			name:    "private 172.16.x IP with HTTPS",
			url:     "https://172.16.0.1/v1",
			wantErr: true,
		},
		{
			name:    "private 192.168.x IP with HTTPS",
			url:     "https://192.168.1.1/v1",
			wantErr: true,
		},
		{
			name:    "loopback 127.x IP (non-127.0.0.1) with HTTPS",
			url:     "https://127.0.0.2/v1",
			wantErr: true,
		},
		{
			name:    "link-local 169.254.x IP with HTTPS",
			url:     "https://169.254.1.1/v1",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			url:     "://not-a-url",
			wantErr: true,
		},
		{
			name:    "empty host",
			url:     "https:///v1",
			wantErr: true,
		},
		{
			name:    "HTTP scheme to non-localhost",
			url:     "http://example.com/v1",
			wantErr: true,
		},
		{
			name:    "HTTP scheme to IP address",
			url:     "http://8.8.8.8/v1",
			wantErr: true,
		},
		{
			name:    "HTTPS to localhost is allowed",
			url:     "https://localhost/v1",
			wantErr: false,
		},
		{
			name:    "HTTP to localhost is allowed",
			url:     "http://localhost/v1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := APIURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("APIURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
