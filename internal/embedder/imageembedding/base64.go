// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package imageembedding

import "encoding/base64"

func base64Encode(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
