// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks_test

import (
	"testing"

	. "codeberg.org/readeck/readeck/internal/testing" //revive:disable:dot-imports
)

//nolint:gocyclo
func TestPermissions(t *testing.T) {
	app := NewTestApp(t)
	defer func() {
		app.Close(t)
	}()

	client := NewClient(t, app)

	users := []string{"admin", "staff", "user", "disabled", ""}
	for _, user := range users {
		RunRequestSequence(t, client, user,
			// API
			RequestTest{
				Target: "/api/bookmarks",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Method: "POST",
				Target: "/api/bookmarks",
				JSON:   map[string]string{},
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 422)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Target: "/api/bookmarks/{{(index .User.Bookmarks 0).UID}}",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Method: "PATCH",
				Target: "/api/bookmarks/{{(index .User.Bookmarks 0).UID}}",
				JSON:   map[string]any{},
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Target: "/api/bookmarks/{{(index .User.Bookmarks 0).UID}}/article",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Target: "/api/bookmarks/{{(index .User.Bookmarks 0).UID}}/x/props.json",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Target: "/api/bookmarks/{{(index .User.Bookmarks 0).UID}}/article.epub",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Target: "/api/bookmarks/{{(index .User.Bookmarks 0).UID}}/article.md",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Target: "/api/bookmarks/annotations",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Target: "/api/bookmarks/collections",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Method: "POST",
				Target: "/api/bookmarks/collections",
				JSON:   true,
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 422)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Target: "/api/bookmarks/collections/RuXBpzio59ktWTEHDodLPU",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 404)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Method: "PATCH",
				Target: "/api/bookmarks/collections/RuXBpzio59ktWTEHDodLPU",
				JSON:   true,
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 404)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Method: "DELETE",
				Target: "/api/bookmarks/collections/RuXBpzio59ktWTEHDodLPU",
				JSON:   true,
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 404)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Target: "/api/bookmarks/labels",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 200)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
			RequestTest{
				Target: "/api/bookmarks/labels/foo",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin", "staff", "user":
						r.AssertStatus(t, 404)
					case "disabled":
						r.AssertStatus(t, 403)
					case "":
						r.AssertStatus(t, 401)
					}
				},
			},
		)
	}
}
