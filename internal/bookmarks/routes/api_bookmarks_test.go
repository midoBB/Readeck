// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package routes_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "codeberg.org/readeck/readeck/internal/testing" //revive:disable:dot-imports
)

func TestBookmarkAPIShare(t *testing.T) {
	app := NewTestApp(t)
	defer func() {
		app.Close(t)
	}()

	client := NewClient(t, app)

	publicPath := ""

	RunRequestSequence(t, client, "user",
		RequestTest{
			Method:       "GET",
			Target:       "/api/bookmarks/{{(index .User.Bookmarks 0).UID}}/share/link",
			JSON:         true,
			ExpectStatus: 201,
			Assert: func(_ *testing.T, r *Response) {
				publicPath = r.Redirect
			},
		},
	)

	require.NotEmpty(t, publicPath, "public path is set")

	RunRequestSequence(t, client, "",
		RequestTest{
			Target:         publicPath,
			ExpectStatus:   200,
			ExpectContains: `Shared by user`,
		},
	)
}
