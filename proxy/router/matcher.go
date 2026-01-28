// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// RouteCondition defines the type of condition to match.
type RouteCondition string

const (
	RouteConditionPath       RouteCondition = "path"
	RouteConditionMethod     RouteCondition = "method"
	RouteConditionHeader     RouteCondition = "header"
	RouteConditionBodyField  RouteCondition = "body_field"
	RouteConditionBodyRegex  RouteCondition = "body_regex"
	RouteConditionQueryParam RouteCondition = "query_param"
)

// RouteMatcher defines a single matching condition (for configuration).
type RouteMatcher struct {
	Condition RouteCondition `json:"condition"`
	Field     string         `json:"field,omitempty"`    // For headers, query params, or body fields
	Pattern   string         `json:"pattern"`            // Pattern to match (can be regex or exact)
	IsRegex   bool           `json:"is_regex,omitempty"` // Whether pattern is a regex
}

// Matcher defines the interface for request matching logic.
type Matcher interface {
	Match(r *http.Request) bool
}

// PathMatcher matches against the request path.
type PathMatcher struct {
	Pattern string
	IsRegex bool
}

func (m *PathMatcher) Match(r *http.Request) bool {
	matched := matchString(r.URL.Path, m.Pattern, m.IsRegex)

	return matched
}

// MethodMatcher matches against the HTTP method.
type MethodMatcher struct {
	Pattern string
	IsRegex bool
}

func (m *MethodMatcher) Match(r *http.Request) bool {
	return matchString(r.Method, m.Pattern, m.IsRegex)
}

// HeaderMatcher matches against a specific header value.
type HeaderMatcher struct {
	Field   string
	Pattern string
	IsRegex bool
}

func (m *HeaderMatcher) Match(r *http.Request) bool {
	headerValue := r.Header.Get(m.Field)
	matched := matchString(headerValue, m.Pattern, m.IsRegex)

	return matched
}

// QueryParamMatcher matches against a query parameter value.
type QueryParamMatcher struct {
	Field   string
	Pattern string
	IsRegex bool
}

func (m *QueryParamMatcher) Match(r *http.Request) bool {
	paramValue := r.URL.Query().Get(m.Field)

	return matchString(paramValue, m.Pattern, m.IsRegex)
}

// BodyFieldMatcher matches against a specific field in the request body (JSON).
type BodyFieldMatcher struct {
	Field   string
	Pattern string
	IsRegex bool
}

func (m *BodyFieldMatcher) Match(r *http.Request) bool {
	bodyBytes, err := readRequestBody(r)
	if err != nil || len(bodyBytes) == 0 {
		return false
	}

	return matchBodyField(bodyBytes, m.Field, m.Pattern, m.IsRegex)
}

// BodyRegexMatcher matches against the entire request body using regex.
type BodyRegexMatcher struct {
	Pattern string
}

func (m *BodyRegexMatcher) Match(r *http.Request) bool {
	bodyBytes, err := readRequestBody(r)
	if err != nil || len(bodyBytes) == 0 {
		return false
	}

	return matchString(string(bodyBytes), m.Pattern, true)
}

// CompositeMatcher allows combining multiple matchers with AND logic.
type CompositeMatcher struct {
	Matchers []Matcher
}

func (m *CompositeMatcher) Match(r *http.Request) bool {
	if len(m.Matchers) == 0 {
		return false
	}

	for _, matcher := range m.Matchers {
		if !matcher.Match(r) {
			return false
		}
	}

	return true
}

func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	// Restore the body so it can be read again
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	return bodyBytes, nil
}

func matchString(value, pattern string, isRegex bool) bool {
	if isRegex {
		matched, err := regexp.MatchString(pattern, value)
		if err != nil {
			return false
		}

		return matched
	}

	return value == pattern
}

func matchBodyField(bodyBytes []byte, field, pattern string, isRegex bool) bool {
	var bodyMap map[string]any

	err := json.Unmarshal(bodyBytes, &bodyMap)
	if err != nil {
		// If it's not JSON, try to match the field as a substring
		bodyStr := string(bodyBytes)
		if strings.Contains(bodyStr, field) {
			return matchString(bodyStr, pattern, isRegex)
		}

		return false
	}

	// Navigate nested fields using dot notation (e.g., "user.profile.name")
	fieldParts := strings.Split(field, ".")

	var current any = bodyMap

	for _, part := range fieldParts {
		switch v := current.(type) {
		case map[string]any:
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

	return matchString(fieldValue, pattern, isRegex)
}

// CreateMatcher creates matchers from RouteMatcher configuration.
func CreateMatcher(rm RouteMatcher) Matcher {
	switch rm.Condition {
	case RouteConditionPath:
		return &PathMatcher{
			Pattern: rm.Pattern,
			IsRegex: rm.IsRegex,
		}
	case RouteConditionMethod:
		return &MethodMatcher{
			Pattern: rm.Pattern,
			IsRegex: rm.IsRegex,
		}
	case RouteConditionHeader:
		return &HeaderMatcher{
			Field:   rm.Field,
			Pattern: rm.Pattern,
			IsRegex: rm.IsRegex,
		}
	case RouteConditionQueryParam:
		return &QueryParamMatcher{
			Field:   rm.Field,
			Pattern: rm.Pattern,
			IsRegex: rm.IsRegex,
		}
	case RouteConditionBodyField:
		return &BodyFieldMatcher{
			Field:   rm.Field,
			Pattern: rm.Pattern,
			IsRegex: rm.IsRegex,
		}
	case RouteConditionBodyRegex:
		return &BodyRegexMatcher{
			Pattern: rm.Pattern,
		}
	default:
		log.Printf("Unknown route condition: %s", rm.Condition)

		return nil
	}
}

// CreateCompositeMatcher creates a composite matcher from multiple RouteMatcher configurations.
func CreateCompositeMatcher(matchers []RouteMatcher) Matcher {
	var compositeMatchers []Matcher

	for _, rm := range matchers {
		if matcher := CreateMatcher(rm); matcher != nil {
			compositeMatchers = append(compositeMatchers, matcher)
		}
	}

	return &CompositeMatcher{
		Matchers: compositeMatchers,
	}
}
