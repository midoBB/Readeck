// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package csp_test

import (
	"net/http"
	"testing"

	"codeberg.org/readeck/readeck/pkg/http/csp"
	"github.com/stretchr/testify/require"
)

func TestCSP(t *testing.T) {
	assert := require.New(t)

	p := csp.Policy{}

	assert.Equal("", p.String())

	p.Add("default-src", csp.None, csp.Self)
	assert.Equal("default-src 'none' 'self'", p.String())

	p.Add("default-src", "example.net")
	assert.Equal("default-src 'none' 'self' example.net", p.String())

	p.Set("default-src", "'self'")
	assert.Equal("default-src 'self'", p.String())

	p.Set("script-src", "'nonce-abcd'", csp.UnsafeInline)
	assert.Equal("default-src 'self'; script-src 'nonce-abcd' 'unsafe-inline'", p.String())

	header := http.Header{}
	p.Write(header)
	assert.Equal(
		"default-src 'self'; script-src 'nonce-abcd' 'unsafe-inline'",
		header.Get("Content-Security-Policy"),
	)

	p2 := p.Clone()
	p2.Add("img-src", csp.Self)
	assert.Equal(
		"default-src 'self'; script-src 'nonce-abcd' 'unsafe-inline'",
		p.String(),
	)
	assert.Equal(
		"default-src 'self'; img-src 'self'; script-src 'nonce-abcd' 'unsafe-inline'",
		p2.String(),
	)

	nonce := csp.MakeNonce()
	assert.Len(nonce, 32)
}
