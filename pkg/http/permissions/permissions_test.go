// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package permissions_test

import (
	"net/http"
	"testing"

	"codeberg.org/readeck/readeck/pkg/http/permissions"
	"github.com/stretchr/testify/require"
)

func TestPP(t *testing.T) {
	assert := require.New(t)

	p := permissions.Policy{}
	assert.Equal("", p.String())

	p.Add(permissions.Microphone, "*")
	assert.Equal(`microphone=(*)`, p.String())

	p.Add(permissions.Microphone, `"http://example.net"`)
	assert.Equal(`microphone=(* "http://example.net")`, p.String())

	p.Set(permissions.Accelerometer)
	assert.Equal(`accelerometer=(), microphone=(* "http://example.net")`, p.String())

	p.Set(permissions.Microphone)
	assert.Equal(`accelerometer=(), microphone=()`, p.String())

	header := http.Header{}
	p.Write(header)
	assert.Equal(
		`accelerometer=(), microphone=()`,
		header.Get("Permissions-Policy"),
	)
}
