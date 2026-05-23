package middleware

import (
	"net/http"
	"strings"
)

const (
	corsAllowedMethods = "GET, POST, PUT, PATCH, OPTIONS"
	corsAllowedHeaders = "Content-Type"
)

type corsConfig struct {
	allowAll       bool
	allowedOrigins map[string]struct{}
}

func CORS(next http.Handler, allowedOriginsValue string) http.Handler {
	config := newCORSConfig(allowedOriginsValue)
	if !config.allowAll && len(config.allowedOrigins) == 0 {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedOrigin, allowed := config.allowedOrigin(origin)

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", corsAllowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			if !allowed {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func newCORSConfig(allowedOriginsValue string) corsConfig {
	config := corsConfig{
		allowedOrigins: map[string]struct{}{},
	}

	for origin := range strings.SplitSeq(allowedOriginsValue, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			config.allowAll = true
			return config
		}

		config.allowedOrigins[origin] = struct{}{}
	}

	return config
}

func (config corsConfig) allowedOrigin(origin string) (string, bool) {
	if origin == "" {
		return "", false
	}
	if config.allowAll {
		return "*", true
	}

	if _, ok := config.allowedOrigins[origin]; ok {
		return origin, true
	}

	return "", false
}
