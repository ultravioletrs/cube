// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package api

import "github.com/absmach/supermq/api/http/util"

type identifyRequest struct {
	Token string `json:"token"`
}

func (i *identifyRequest) Validate() error {
	if i.Token == "" {
		return util.ErrBearerToken
	}

	return nil
}
