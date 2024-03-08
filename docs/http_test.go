// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package docs_test

import (
	"testing"

	. "codeberg.org/readeck/readeck/internal/testing" //revive:disable:dot-imports
	"github.com/stretchr/testify/require"
)

func TestDocs(t *testing.T) {
	app := NewTestApp(t)
	defer func() {
		app.Close(t)
	}()

	client := NewClient(t, app)
	users := []string{"admin", "staff", "user", "disabled", ""}
	for _, user := range users {
		RunRequestSequence(t, client, user,
			RequestTest{
				Target: "/docs",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/docs/en-US/")
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/login")
					}
				},
			},
			RequestTest{
				Target: "/docs/bookmark",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/docs/en-US/bookmark")
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/login")
					}
				},
			},
			RequestTest{
				Target: "/docs/en-US/",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/login")
					}
				},
			},
			RequestTest{
				Target: "/docs/en-US/bookmark",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/login")
					}
				},
			},
			RequestTest{
				Target:       "/docs/en-US/not-found",
				ExpectStatus: 404,
			},
			RequestTest{
				Target: "/docs/en-US/img/bookmark-new.webp",
				Assert: func(t *testing.T, r *Response) {
					r.AssertStatus(t, 200)
					require.Equal(t, "image/webp", r.Header.Get("content-type"))
				},
			},
			RequestTest{
				Target: "/docs/about",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "statff":
						r.AssertStatus(t, 200)
					case "user", "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/login")
					}
				},
			},
			RequestTest{
				Target: "/docs/api",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/login")
					}
				},
			},
			RequestTest{
				Target: "/docs/api.json",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
						require.Contains(t, r.Header.Get("content-type"), "application/json")
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/login")
					}
				},
			},
		)
	}
}
