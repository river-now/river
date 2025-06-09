package netutil

import (
	"net/http"
	"testing"
)

// TestIsLocalhost tests the isLocalhost function
func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected bool
	}{
		// Localhost variations
		{"localhost without port", "localhost", true},
		{"localhost with port", "localhost:8080", true},
		{"LOCALHOST uppercase", "LOCALHOST", true},
		{"Localhost mixed case", "Localhost:3000", true},

		// IPv4 loopback addresses
		{"127.0.0.1 without port", "127.0.0.1", true},
		{"127.0.0.1 with port", "127.0.0.1:8080", true},
		{"127.0.0.2", "127.0.0.2:80", true},
		{"127.255.255.255", "127.255.255.255", true},
		{"127.1.2.3", "127.1.2.3:443", true},

		// IPv6 loopback addresses
		{"IPv6 ::1", "::1", true},
		{"IPv6 [::1]", "[::1]", true},
		{"IPv6 ::1 with port", "[::1]:8080", true},
		{"IPv6 long form", "0:0:0:0:0:0:0:1", true},
		{"IPv6 long form with brackets", "[0:0:0:0:0:0:0:1]", true},
		{"IPv6 long form with port", "[0:0:0:0:0:0:0:1]:80", true},
		{"IPv4-mapped IPv6", "::ffff:127.0.0.1", true},
		{"IPv4-mapped IPv6 with brackets", "[::ffff:127.0.0.1]", true},
		{"IPv4-mapped IPv6 with port", "[::ffff:127.0.0.1]:8080", true},

		// Production hosts - MUST return false
		{"example.com", "example.com", false},
		{"example.com with port", "example.com:80", false},
		{"www.google.com", "www.google.com", false},
		{"api.production.com", "api.production.com:443", false},
		{"myapp.herokuapp.com", "myapp.herokuapp.com", false},

		// Private IP addresses - should return false
		{"Private 10.x", "10.0.0.1", false},
		{"Private 172.16.x", "172.16.0.1:8080", false},
		{"Private 192.168.x", "192.168.1.1", false},

		// Public IP addresses - should return false
		{"Google DNS", "8.8.8.8", false},
		{"Cloudflare DNS", "1.1.1.1:53", false},
		{"Random public IP", "93.184.216.34", false},

		// Edge cases
		{"Empty string", "", false},
		{"Just port", ":8080", false},
		{"Invalid IP", "256.256.256.256", false},
		{"Partial localhost", "local", false},
		{"localhost prefix", "localhost.com", false},
		{"localhost suffix", "mylocalhost", false},
		{"127.0.0.1 prefix", "127.0.0.1.com", false},
		{"Almost loopback", "128.0.0.1", false},
		{"IPv6 private", "fe80::1", false},
		{"IPv6 public", "2001:db8::1", false},

		// Malformed inputs
		{"Multiple brackets", "[[[::1]]]", false},
		{"Unmatched brackets", "[::1", false},
		{"Wrong bracket order", "]::1[", false},
		{"Spaces in host", "127.0.0.1 ", false},
		{"URL instead of host", "http://localhost", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Host: tt.host,
			}
			result := IsLocalhost(req.Host)
			if result != tt.expected {
				t.Errorf("isLocalhost(%q) = %v, want %v", tt.host, result, tt.expected)
			}
		})
	}
}

// Test specifically for production-like hostnames to ensure they never pass
func TestProductionHostsSafety(t *testing.T) {
	productionHosts := []string{
		// Common production domains
		"mycompany.com",
		"api.mycompany.com",
		"app.mycompany.io",
		"service.company.net",
		"production.internal",
		"prod.example.com",

		// Cloud providers
		"myapp.herokuapp.com",
		"myfunction.azurewebsites.net",
		"myapp.us-east-1.elb.amazonaws.com",
		"myproject.appspot.com",
		"myapp.vercel.app",
		"myapp.netlify.app",

		// With various ports
		"production.com:443",
		"api.company.com:8080",
		"secure.bank.com:443",

		// Tricky variations
		"localhost.mydomain.com",
		"127.0.0.1.mydomain.com",
		"fake-localhost.com",
		"notlocalhost",
		"localhost123",
		"127001",
	}

	for _, host := range productionHosts {
		t.Run("Production: "+host, func(t *testing.T) {
			req := &http.Request{Host: host}
			if IsLocalhost(req.Host) {
				t.Errorf("CRITICAL: isLocalhost(%q) returned true for production host!", host)
			}
		})
	}
}
