// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package permissionspolicy_test

import (
	"net/http"
	"testing"

	"codeberg.org/readeck/readeck/pkg/http/permissionspolicy"
	"github.com/stretchr/testify/require"
)

func TestPP(t *testing.T) {
	assert := require.New(t)

	p := permissionspolicy.Policy{}
	assert.Equal("", p.String())

	p.Add(permissionspolicy.Microphone, "*")
	assert.Equal(`microphone=(*)`, p.String())

	p.Add(permissionspolicy.Microphone, `"http://example.net"`)
	assert.Equal(`microphone=(* "http://example.net")`, p.String())

	p.Set(permissionspolicy.Accelerometer)
	assert.Equal(`accelerometer=(), microphone=(* "http://example.net")`, p.String())

	p.Set(permissionspolicy.Microphone)
	assert.Equal(`accelerometer=(), microphone=()`, p.String())

	header := http.Header{}
	p.Write(header)
	assert.Equal(
		`accelerometer=(), microphone=()`,
		header.Get("Permissions-Policy"),
	)
}
