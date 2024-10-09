// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package api

import (
	"net/http"

	"github.com/absmach/magistrala"
)

var _ magistrala.Response = (*identifyResponse)(nil)

type identifyResponse struct {
	identified bool
}

func (i *identifyResponse) Code() int {
	if i.identified {
		return http.StatusOK
	}

	return http.StatusUnauthorized
}

func (i *identifyResponse) Headers() map[string]string {
	return map[string]string{}
}

func (i identifyResponse) Empty() bool {
	return true
}
