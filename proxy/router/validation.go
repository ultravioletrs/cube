// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package router

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// System routes that cannot be deleted or modified.
var systemRoutes = map[string]bool{
	"attestation": true,
	"health":      true,
}

// ValidateRouteName checks if a route name is valid.
// Valid names contain only alphanumeric characters, hyphens, and underscores.
func ValidateRouteName(name string) error {
	if name == "" {
		return ErrRouteNameRequired
	}

	// Check length (1-255 characters)
	if len(name) > 255 {
		return fmt.Errorf("%w: name too long (max 255 characters)", ErrInvalidRouteName)
	}

	// Check format: alphanumeric, hyphens, underscores only
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return ErrInvalidRouteName
	}

	return nil
}

// ValidateURL checks if a target URL is valid.
func ValidateURL(targetURL string) error {
	if targetURL == "" {
		return fmt.Errorf("%w: target URL is required", ErrInvalidURL)
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	// Ensure URL has a scheme
	if parsedURL.Scheme == "" {
		return fmt.Errorf("%w: URL must include scheme (http/https)", ErrInvalidURL)
	}

	// Ensure URL has a host
	if parsedURL.Host == "" {
		return fmt.Errorf("%w: URL must include host", ErrInvalidURL)
	}

	return nil
}

// ValidateMatchers checks if route matchers are valid.
func ValidateMatchers(matchers []RouteMatcher) error {
	for i, matcher := range matchers {
		if err := ValidateMatcher(matcher); err != nil {
			return fmt.Errorf("matcher %d: %w", i, err)
		}
	}

	return nil
}

// ValidateMatcher checks if a single route matcher is valid.
func ValidateMatcher(matcher RouteMatcher) error {
	// Validate condition type
	validConditions := map[RouteCondition]bool{
		RouteConditionPath:       true,
		RouteConditionMethod:     true,
		RouteConditionHeader:     true,
		RouteConditionBodyField:  true,
		RouteConditionBodyRegex:  true,
		RouteConditionQueryParam: true,
	}

	if !validConditions[matcher.Condition] {
		return fmt.Errorf("%w: unknown condition type '%s'", ErrInvalidMatcher, matcher.Condition)
	}

	// Validate pattern is not empty
	if matcher.Pattern == "" {
		return fmt.Errorf("%w: pattern cannot be empty", ErrInvalidMatcher)
	}

	// Validate field is present for conditions that require it
	fieldRequiredConditions := map[RouteCondition]bool{
		RouteConditionHeader:     true,
		RouteConditionBodyField:  true,
		RouteConditionQueryParam: true,
	}

	if fieldRequiredConditions[matcher.Condition] && matcher.Field == "" {
		return fmt.Errorf("%w: field is required for condition '%s'", ErrInvalidMatcher, matcher.Condition)
	}

	// Validate regex pattern if IsRegex is true
	if matcher.IsRegex {
		if _, err := regexp.Compile(matcher.Pattern); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidRegex, err)
		}
	}

	return nil
}

// ValidatePriority checks if priority is within valid range.
func ValidatePriority(priority int) error {
	if priority < 0 || priority > 1000 {
		return ErrInvalidPriority
	}

	return nil
}

// ValidateRoute performs comprehensive validation on a route rule.
func ValidateRoute(route *RouteRule) error {
	// Validate name
	if err := ValidateRouteName(route.Name); err != nil {
		return err
	}

	// Validate target URL
	if err := ValidateURL(route.TargetURL); err != nil {
		return err
	}

	// Validate priority
	if err := ValidatePriority(route.Priority); err != nil {
		return err
	}

	// Validate matchers (if not a default rule)
	if !route.DefaultRule {
		if len(route.Matchers) == 0 {
			return ErrNoMatchers
		}

		if err := ValidateMatchers(route.Matchers); err != nil {
			return err
		}
	}

	// Validate strip_prefix format if present
	if route.StripPrefix != "" {
		// Strip prefix should start with /
		if !strings.HasPrefix(route.StripPrefix, "/") {
			return fmt.Errorf("strip_prefix must start with /")
		}
	}

	return nil
}

// IsSystemRoute checks if a route is a protected system route.
func IsSystemRoute(name string) bool {
	return systemRoutes[name]
}

// DetectConflict checks if a new route conflicts with existing routes.
// This is a basic implementation that checks for exact name conflicts.
// More sophisticated conflict detection could check for overlapping matchers.
func DetectConflict(newRoute *RouteRule, existingRoutes []RouteRule) error {
	// Check for duplicate names
	for _, existing := range existingRoutes {
		if existing.Name == newRoute.Name {
			return fmt.Errorf("%w: route with name '%s' already exists", ErrRouteConflict, newRoute.Name)
		}
	}

	// Check for multiple default routes
	if newRoute.DefaultRule {
		for _, existing := range existingRoutes {
			if existing.DefaultRule {
				return fmt.Errorf("%w: default route '%s' already exists", ErrMultipleDefaultRoutes, existing.Name)
			}
		}
	}

	return nil
}
