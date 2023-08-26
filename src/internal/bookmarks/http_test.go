// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks_test

import (
	"testing"

	. "github.com/readeck/readeck/internal/testing"
)

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
				JSON:   true,
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
				JSON:   true,
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
				JSON:   true,
				Method: "POST",
				Target: "/api/bookmarks/collections",
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
				JSON:   true,
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
				JSON:   true,
				Method: "PATCH",
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
				JSON:   true,
				Method: "DELETE",
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
		)
	}
}
