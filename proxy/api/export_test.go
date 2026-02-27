// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

//nolint:gochecknoglobals // export_test.go aliases for white-box testing of unexported functions.
var (
	InjectAuditFilter          = injectAuditFilter
	InjectAuditFilterIntoQuery = injectAuditFilterIntoQuery
	InjectAuditFilterIntoBody  = injectAuditFilterIntoBody
)
