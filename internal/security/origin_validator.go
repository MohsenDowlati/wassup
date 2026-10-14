package security

import (
	"net/http"
	"net/url"
	"strings"
)

type OriginValidator struct {
	allowedOrigins map[string]struct{}
	allowAnyLocal  bool
}

func NewOriginValidator(allowedHosts []string, allowLocalhost bool) *OriginValidator {
	originMap := make(map[string]struct{}, len(allowedHosts))
	for _, h := range allowedHosts {
		originMap[strings.ToLower(strings.TrimSpace(h))] = struct{}{}
	}
	return &OriginValidator{
		allowedOrigins: originMap,
		allowAnyLocal:  allowLocalhost,
	}
}

func (v *OriginValidator) IsOriginAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	host := strings.ToLower(u.Host)

	if v.allowAnyLocal && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1") {
		return true
	}

	_, exists := v.allowedOrigins[host]
	return exists
}
