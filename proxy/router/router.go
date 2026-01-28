// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package router

import (
	"errors"
	"net/http"
	"sort"
	"sync"
)

// ErrNoMatchingRoute indicates that no route matched the request.
var ErrNoMatchingRoute = errors.New("no matching route found")

// RouteRule defines a complete routing rule.
type RouteRule struct {
	Name        string         `json:"name"`
	TargetURL   string         `json:"target_url"`
	Matchers    []RouteMatcher `json:"matchers"`               // All matchers must match (AND logic)
	Priority    int            `json:"priority"`               // Higher priority rules are checked first
	DefaultRule bool           `json:"default_rule"`           // If true, this rule matches when no others do
	StripPrefix string         `json:"strip_prefix,omitempty"` // Prefix to strip from the path

	// Internal compiled matcher (not serialized)
	compiledMatcher Matcher `json:"-"`
}

// RouteRules implements sort.Interface for []RouteRule based on Priority field.
// Routes are sorted in descending order of priority (higher priority first).
type RouteRules []RouteRule

func (r RouteRules) Len() int           { return len(r) }
func (r RouteRules) Less(i, j int) bool { return r[i].Priority > r[j].Priority } // Descending order
func (r RouteRules) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

// Config contains the routing configuration.
type Config struct {
	Routes     []RouteRule `json:"routes"`
	DefaultURL string      `json:"default_url,omitempty"` // Fallback if no default rule is defined
}

type Router struct {
	mu         sync.RWMutex
	routes     RouteRules
	defaultURL string
}

func New(config Config) *Router {
	routes := make(RouteRules, len(config.Routes))
	copy(routes, config.Routes)

	for i := range routes {
		routes[i].compiledMatcher = CreateCompositeMatcher(routes[i].Matchers)
	}

	sort.Sort(routes)

	return &Router{
		routes:     routes,
		defaultURL: config.DefaultURL,
	}
}

// UpdateRoutes atomically replaces the current routes with new ones.
// This allows runtime modification of routes without restart.
func (r *Router) UpdateRoutes(newRoutes []RouteRule) {
	r.mu.Lock()
	defer r.mu.Unlock()

	routes := make(RouteRules, len(newRoutes))
	copy(routes, newRoutes)

	// Compile matchers for all routes
	for i := range routes {
		routes[i].compiledMatcher = CreateCompositeMatcher(routes[i].Matchers)
	}

	// Sort by priority (descending)
	sort.Sort(routes)

	r.routes = routes
}

func (r *Router) DetermineTarget(req *http.Request) (targetURL, stripPrefix string, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, route := range r.routes {
		if route.DefaultRule {
			continue // Skip default rules in main loop
		}

		if route.compiledMatcher != nil && route.compiledMatcher.Match(req) {
			return route.TargetURL, route.StripPrefix, nil
		}
	}

	for _, route := range r.routes {
		if route.DefaultRule {
			return route.TargetURL, route.StripPrefix, nil
		}
	}

	if r.defaultURL != "" {
		return r.defaultURL, "", nil
	}

	return "", "", ErrNoMatchingRoute
}
