// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package router

import (
	"errors"
	"log"
	"net/http"
	"sort"
)

var (
	// ErrNoMatchingRoute indicates that no route matched the request.
	ErrNoMatchingRoute = errors.New("no matching route found")
	// errRouteNotFound indicates that the specified route was not found.
	errRouteNotFound = errors.New("route not found")
)

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

// RouterConfig contains the routing configuration.
type RouterConfig struct {
	Routes     []RouteRule `json:"routes"`
	DefaultURL string      `json:"default_url,omitempty"` // Fallback if no default rule is defined
}

type Router struct {
	routes     RouteRules
	defaultURL string
}

func New(config RouterConfig) *Router {
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

func (r *Router) DetermineTarget(req *http.Request) (string, string, error) {
	for _, route := range r.routes {
		if route.DefaultRule {
			continue // Skip default rules in main loop
		}

		if route.compiledMatcher != nil && route.compiledMatcher.Match(req) {
			log.Printf("Request matched route: %s -> %s", route.Name, route.TargetURL)

			return route.TargetURL, route.StripPrefix, nil
		}
	}

	for _, route := range r.routes {
		if route.DefaultRule {
			log.Printf("Request matched default route: %s -> %s", route.Name, route.TargetURL)

			return route.TargetURL, route.StripPrefix, nil
		}
	}

	if r.defaultURL != "" {
		log.Printf("Request using config default URL: %s", r.defaultURL)

		return r.defaultURL, "", nil
	}

	return "", "", ErrNoMatchingRoute
}
