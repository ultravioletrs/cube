// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package router

import "errors"

// Route validation and management errors.
var (
	ErrInvalidRouteName      = errors.New("invalid route name: must be alphanumeric with hyphens or underscores")
	ErrRouteNameRequired     = errors.New("route name is required")
	ErrInvalidURL            = errors.New("invalid target URL")
	ErrInvalidMatcher        = errors.New("invalid route matcher configuration")
	ErrInvalidRegex          = errors.New("invalid regex pattern in matcher")
	ErrRouteConflict         = errors.New("route conflicts with existing route")
	ErrSystemRouteProtected  = errors.New("system route cannot be modified or deleted")
	ErrRouteNotFound         = errors.New("route not found")
	ErrInvalidPriority       = errors.New("invalid priority: must be between 0 and 1000")
	ErrNoMatchers            = errors.New("route must have at least one matcher or be a default rule")
	ErrMultipleDefaultRoutes = errors.New("only one default route is allowed")
)
