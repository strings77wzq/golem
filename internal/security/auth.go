// Package security provides HTTP middleware for authentication (Bearer token
// validation), per-IP rate limiting, and a command sandbox that restricts
// which shell commands the exec tool may run. All three concerns are
// composable net/http middleware and can be applied independently.
package security

import (
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

// AuthConfig holds configuration for API key authentication middleware
type AuthConfig struct {
	APIKeys        []string // valid API keys
	AllowFrom      []string // allowed IP CIDRs (empty = allow all)
	TrustedProxies []string // proxy IPs allowed to set X-Forwarded-For (empty = trust none)
	Enabled        bool     // if false, skip auth
}

// AuthMiddleware returns HTTP middleware that validates API keys
func AuthMiddleware(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If disabled, pass through
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Extract API key from X-API-Key header or Authorization Bearer token
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					apiKey = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			// Check if API key is valid
			if apiKey == "" || !isValidAPIKey(apiKey, cfg.APIKeys) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"}) //nolint:errcheck
				return
			}

			// If AllowFrom is set, validate source IP
			if len(cfg.AllowFrom) > 0 {
				clientIP := getClientIP(r, cfg.TrustedProxies)
				if !isIPAllowed(clientIP, cfg.AllowFrom) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"}) //nolint:errcheck
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isValidAPIKey checks if the provided key is in the list of valid keys.
// Uses constant-time comparison to prevent timing side-channel attacks.
func isValidAPIKey(key string, validKeys []string) bool {
	for _, validKey := range validKeys {
		if subtle.ConstantTimeCompare([]byte(key), []byte(validKey)) == 1 {
			return true
		}
	}
	return false
}

// getClientIP extracts the client IP from the request. Proxy headers
// (X-Forwarded-For / X-Real-IP) are trusted only when the direct peer is in
// the trustedProxies allowlist; otherwise they are attacker-controlled and
// must not drive rate limiting or IP allowlisting. Empty trustedProxies
// (the default) means "trust no proxy".
func getClientIP(r *http.Request, trustedProxies []string) string {
	if isTrustedProxy(r.RemoteAddr, trustedProxies) {
		// Try X-Forwarded-For first
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				return strings.TrimSpace(ips[0])
			}
		}

		// Try X-Real-IP
		xri := r.Header.Get("X-Real-IP")
		if xri != "" {
			return xri
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// isTrustedProxy reports whether the direct peer (RemoteAddr host) is in the
// trusted proxy allowlist. When empty, no proxy is trusted.
func isTrustedProxy(remoteAddr string, trusted []string) bool {
	if len(trusted) == 0 {
		return false
	}
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	for _, tp := range trusted {
		if host == tp {
			return true
		}
	}
	return false
}

// isIPAllowed checks if the client IP is in the allowed CIDR ranges
func isIPAllowed(clientIP string, allowedCIDRs []string) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	for _, cidr := range allowedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
