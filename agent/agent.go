// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/ultravioletrs/cocos/pkg/attestation"
	"github.com/ultravioletrs/cocos/pkg/attestation/quoteprovider"
	"github.com/ultravioletrs/cocos/pkg/attestation/vtpm"
)

var (
	// ErrAttestationFailed indicates that attestation failed.
	ErrAttestationFailed = errors.New("attestation failed")
	// ErrAttestationVTpmFailed indicates that vTPM attestation failed.
	ErrAttestationVTpmFailed = errors.New("vTPM attestation failed")
	// ErrAttestationType indicates that the attestation type is invalid.
	ErrAttestationType = errors.New("invalid attestation type")
	// ErrUnauthorized indicates that authentication failed.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrNoMatchingRoute indicates that no route matched the request.
	ErrNoMatchingRoute = errors.New("no matching route found")
)

type TLSConfig struct {
	Enabled            bool
	InsecureSkipVerify bool
	CertFile           string
	KeyFile            string
	CAFile             string
	MinVersion         uint16
	MaxVersion         uint16
}

// RouteCondition defines the type of condition to match
type RouteCondition string

const (
	RouteConditionPath       RouteCondition = "path"
	RouteConditionMethod     RouteCondition = "method"
	RouteConditionHeader     RouteCondition = "header"
	RouteConditionBodyField  RouteCondition = "body_field"
	RouteConditionBodyRegex  RouteCondition = "body_regex"
	RouteConditionQueryParam RouteCondition = "query_param"
)

// RouteMatcher defines a single matching condition
type RouteMatcher struct {
	Condition RouteCondition `json:"condition"`
	Field     string         `json:"field,omitempty"`    // For headers, query params, or body fields
	Pattern   string         `json:"pattern"`            // Pattern to match (can be regex or exact)
	IsRegex   bool           `json:"is_regex,omitempty"` // Whether pattern is a regex
}

// RouteRule defines a complete routing rule
type RouteRule struct {
	Name        string         `json:"name"`
	TargetURL   string         `json:"target_url"`
	Matchers    []RouteMatcher `json:"matchers"` // All matchers must match (AND logic)
	Priority    int            `json:"priority"` // Higher priority rules are checked first
	DefaultRule bool           `json:"default"`  // If true, this rule matches when no others do
}

// RouterConfig contains the routing configuration
type RouterConfig struct {
	Routes     []RouteRule `json:"routes"`
	DefaultURL string      `json:"default_url,omitempty"` // Fallback if no default rule is defined
}

type Config struct {
	Router RouterConfig `json:"router"`
	TLS    TLSConfig    `json:"tls"`
}

type agentService struct {
	config    *Config
	provider  attestation.Provider
	transport *http.Transport
	auth      authn.Authentication
	routes    []RouteRule
}

type Service interface {
	Proxy() http.Handler
	Attestation(
		reportData [quoteprovider.Nonce]byte, nonce [vtpm.Nonce]byte, attType attestation.PlatformType,
	) ([]byte, error)
	Authenticate(req *http.Request) error
	AuthMiddleware(next http.Handler) http.Handler
	AddRoute(rule RouteRule) error
	RemoveRoute(name string) error
	GetRoutes() []RouteRule
}

func New(config *Config, auth authn.Authentication, provider attestation.Provider) (Service, error) {
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	if config.TLS.Enabled {
		tlsConfig, err := setTLSConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to set TLS config: %w", err)
		}

		transport.TLSClientConfig = tlsConfig
	}

	// Sort routes by priority (higher first)
	routes := make([]RouteRule, len(config.Router.Routes))
	copy(routes, config.Router.Routes)
	for i := 0; i < len(routes); i++ {
		for j := i + 1; j < len(routes); j++ {
			if routes[j].Priority > routes[i].Priority {
				routes[i], routes[j] = routes[j], routes[i]
			}
		}
	}

	return &agentService{
		config:    config,
		transport: transport,
		provider:  provider,
		auth:      auth,
		routes:    routes,
	}, nil
}

func (a *agentService) Authenticate(req *http.Request) error {
	token := util.ExtractBearerToken(req)

	if token == "" {
		return errors.Wrap(ErrUnauthorized, errors.New("missing or invalid token"))
	}

	_, err := a.auth.Authenticate(req.Context(), token)
	if err != nil {
		return errors.Wrap(ErrUnauthorized, err)
	}

	return nil
}

func (a *agentService) Proxy() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetURL, err := a.determineTarget(r)
		if err != nil {
			log.Printf("Failed to determine target: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		target, err := url.Parse(targetURL)
		if err != nil {
			log.Printf("Invalid target URL %s: %v", targetURL, err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.Transport = a.transport

		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			a.modifyHeaders(req)
			log.Printf("Agent forwarding to %s: %s %s", targetURL, req.Method, req.URL.Path)
		}

		proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
			log.Printf("Proxy error: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		}

		proxy.ServeHTTP(w, r)
	})
}

func (a *agentService) determineTarget(r *http.Request) (string, error) {
	// Read body once and create a copy for each route check
	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read request body: %w", err)
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// Check routes in priority order
	for _, route := range a.routes {
		if route.DefaultRule {
			continue // Skip default rules in main loop
		}

		if a.matchesRoute(r, bodyBytes, route) {
			log.Printf("Request matched route: %s -> %s", route.Name, route.TargetURL)
			return route.TargetURL, nil
		}
	}

	// Check for default rule
	for _, route := range a.routes {
		if route.DefaultRule {
			log.Printf("Request matched default route: %s -> %s", route.Name, route.TargetURL)
			return route.TargetURL, nil
		}
	}

	// Use config default URL if available
	if a.config.Router.DefaultURL != "" {
		log.Printf("Request using config default URL: %s", a.config.Router.DefaultURL)
		return a.config.Router.DefaultURL, nil
	}

	return "", ErrNoMatchingRoute
}

func (a *agentService) matchesRoute(r *http.Request, bodyBytes []byte, route RouteRule) bool {
	// All matchers must match (AND logic)
	for _, matcher := range route.Matchers {
		if !a.matchesSingleCondition(r, bodyBytes, matcher) {
			return false
		}
	}
	return len(route.Matchers) > 0 // Must have at least one matcher
}

func (a *agentService) matchesSingleCondition(r *http.Request, bodyBytes []byte, matcher RouteMatcher) bool {
	switch matcher.Condition {
	case RouteConditionPath:
		return a.matchString(r.URL.Path, matcher.Pattern, matcher.IsRegex)

	case RouteConditionMethod:
		return a.matchString(r.Method, matcher.Pattern, matcher.IsRegex)

	case RouteConditionHeader:
		headerValue := r.Header.Get(matcher.Field)
		return a.matchString(headerValue, matcher.Pattern, matcher.IsRegex)

	case RouteConditionQueryParam:
		paramValue := r.URL.Query().Get(matcher.Field)
		return a.matchString(paramValue, matcher.Pattern, matcher.IsRegex)

	case RouteConditionBodyField:
		if len(bodyBytes) == 0 {
			return false
		}
		return a.matchBodyField(bodyBytes, matcher.Field, matcher.Pattern, matcher.IsRegex)

	case RouteConditionBodyRegex:
		if len(bodyBytes) == 0 {
			return false
		}
		return a.matchString(string(bodyBytes), matcher.Pattern, true) // Always regex for body_regex

	default:
		log.Printf("Unknown route condition: %s", matcher.Condition)
		return false
	}
}

func (a *agentService) matchString(value, pattern string, isRegex bool) bool {
	if isRegex {
		matched, err := regexp.MatchString(pattern, value)
		if err != nil {
			log.Printf("Invalid regex pattern %s: %v", pattern, err)
			return false
		}
		return matched
	}
	return value == pattern
}

func (a *agentService) matchBodyField(bodyBytes []byte, field, pattern string, isRegex bool) bool {
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		// If it's not JSON, try to match the field as a substring
		bodyStr := string(bodyBytes)
		if strings.Contains(bodyStr, field) {
			return a.matchString(bodyStr, pattern, isRegex)
		}
		return false
	}

	// Navigate nested fields using dot notation (e.g., "user.profile.name")
	fieldParts := strings.Split(field, ".")
	var current interface{} = bodyMap

	for _, part := range fieldParts {
		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[part]
			if !ok {
				return false
			}
		default:
			return false
		}
	}

	// Convert the field value to string for matching
	fieldValue := fmt.Sprintf("%v", current)
	return a.matchString(fieldValue, pattern, isRegex)
}

func (a *agentService) AddRoute(rule RouteRule) error {
	// Validate the rule
	if rule.Name == "" {
		return errors.New("route name is required")
	}
	if rule.TargetURL == "" {
		return errors.New("target URL is required")
	}
	if _, err := url.Parse(rule.TargetURL); err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	// Remove existing route with same name
	a.RemoveRoute(rule.Name)

	// Add new route
	a.routes = append(a.routes, rule)

	// Re-sort by priority
	for i := 0; i < len(a.routes); i++ {
		for j := i + 1; j < len(a.routes); j++ {
			if a.routes[j].Priority > a.routes[i].Priority {
				a.routes[i], a.routes[j] = a.routes[j], a.routes[i]
			}
		}
	}

	return nil
}

func (a *agentService) RemoveRoute(name string) error {
	for i, route := range a.routes {
		if route.Name == name {
			a.routes = append(a.routes[:i], a.routes[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("route %s not found", name)
}

func (a *agentService) GetRoutes() []RouteRule {
	routes := make([]RouteRule, len(a.routes))
	copy(routes, a.routes)
	return routes
}

func (a *agentService) Attestation(
	reportData [quoteprovider.Nonce]byte, nonce [vtpm.Nonce]byte, attType attestation.PlatformType,
) ([]byte, error) {
	switch attType {
	case attestation.SNP, attestation.TDX:
		rawQuote, err := a.provider.TeeAttestation(reportData[:])
		if err != nil {
			return []byte{}, errors.Wrap(ErrAttestationFailed, err)
		}
		return rawQuote, nil
	case attestation.VTPM:
		vTPMQuote, err := a.provider.VTpmAttestation(nonce[:])
		if err != nil {
			return []byte{}, errors.Wrap(ErrAttestationVTpmFailed, err)
		}
		return vTPMQuote, nil
	case attestation.SNPvTPM:
		vTPMQuote, err := a.provider.Attestation(reportData[:], nonce[:])
		if err != nil {
			return []byte{}, errors.Wrap(ErrAttestationVTpmFailed, err)
		}
		return vTPMQuote, nil
	case attestation.Azure, attestation.NoCC:
		return []byte{}, ErrAttestationType
	default:
		return []byte{}, ErrAttestationType
	}
}

func (a *agentService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := a.Authenticate(r)
		if err != nil {
			log.Printf("Authentication failed: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *agentService) modifyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Del("Authorization")
}

func setTLSConfig(config *Config) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.TLS.InsecureSkipVerify,
	}

	if config.TLS.MinVersion != 0 {
		tlsConfig.MinVersion = config.TLS.MinVersion
	}

	if config.TLS.MaxVersion != 0 {
		tlsConfig.MaxVersion = config.TLS.MaxVersion
	}

	if config.TLS.CertFile != "" && config.TLS.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.TLS.CertFile, config.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}
