// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package admin_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	. "codeberg.org/readeck/readeck/internal/testing" //revive:disable:dot-imports
)

func TestViews(t *testing.T) {
	app := NewTestApp(t)
	defer func() {
		app.Close(t)
	}()

	client := NewClient(t, app)
	u1, err := NewTestUser("test1", "test1@localhost", "test1", "user")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("users", func(t *testing.T) {
		RunRequestSequence(t, client, "admin",
			RequestTest{
				Target:         "/admin",
				ExpectStatus:   303,
				ExpectRedirect: "/admin/users",
			},
			RequestTest{
				Target:         "/admin/users",
				ExpectStatus:   200,
				ExpectContains: "Users</h1>",
			},
			RequestTest{
				Target:         "/admin/users/add",
				ExpectStatus:   200,
				ExpectContains: "New User</h1>",
			},

			// Create user
			RequestTest{
				Method:         "POST",
				Target:         "/admin/users/add",
				Form:           url.Values{},
				ExpectStatus:   422,
				ExpectContains: "Please check your form for errors.",
			},
			RequestTest{
				Target: "/admin/users/add",
			},
			RequestTest{
				Method: "POST",
				Target: "/admin/users/add",
				Form: url.Values{
					"username": {"test3"},
					"password": {"1234"},
					"email":    {"test3@localhost"},
					"group":    {"user"},
				},
				ExpectStatus:   303,
				ExpectRedirect: `^/admin/users/\d+$`,
			},

			// Update user
			RequestTest{
				Target:         fmt.Sprintf("/admin/users/%d", u1.User.ID),
				ExpectStatus:   200,
				ExpectContains: "test1</h1>",
			},
			RequestTest{
				Method:         "POST",
				Target:         fmt.Sprintf("/admin/users/%d", u1.User.ID),
				ExpectStatus:   303,
				ExpectRedirect: fmt.Sprintf("/admin/users/%d", u1.User.ID),
			},
			RequestTest{
				Target:         fmt.Sprintf("/admin/users/%d", u1.User.ID),
				ExpectStatus:   200,
				ExpectContains: "<strong>User updated</strong>",
			},

			// Udpate current user
			RequestTest{
				Target: fmt.Sprintf("/admin/users/%d", app.Users["admin"].User.ID),
			},
			RequestTest{
				Method:         "POST",
				Target:         fmt.Sprintf("/admin/users/%d", app.Users["admin"].User.ID),
				ExpectStatus:   303,
				ExpectRedirect: fmt.Sprintf("/admin/users/%d", app.Users["admin"].User.ID),
			},
			RequestTest{
				Target:         fmt.Sprintf("/admin/users/%d", u1.User.ID),
				ExpectStatus:   200,
				ExpectContains: "<strong>User updated</strong>",
			},

			// Delete user
			RequestTest{
				Target: fmt.Sprintf("/admin/users/%d", u1.User.ID),
			},
			RequestTest{
				Method:         "POST",
				Target:         fmt.Sprintf("/admin/users/%d/delete", u1.User.ID),
				ExpectStatus:   303,
				ExpectRedirect: "/admin/users",
			},
			RequestTest{
				Target:         fmt.Sprintf("/admin/users/%d", u1.User.ID),
				ExpectStatus:   200,
				ExpectContains: "User will be removed in a few seconds",
				Assert: func(t *testing.T, r *Response) {
					assert := require.New(t)
					evt := map[string]interface{}{}

					// An event was sent
					assert.Len(Events().Records("task"), 1)
					assert.NoError(json.Unmarshal(Events().Records("task")[0], &evt))
					assert.Equal("user.delete", evt["name"])
					assert.Equal(float64(u1.User.ID), evt["id"])

					// There's a task in the store
					task := fmt.Sprintf("tasks:user.delete:%d", u1.User.ID)
					m := Store().Get(task)
					assert.NotEmpty(m)
				},
			},

			// Cancel deletion
			RequestTest{
				Target: fmt.Sprintf("/admin/users/%d", u1.User.ID),
			},
			RequestTest{
				Method:         "POST",
				Target:         fmt.Sprintf("/admin/users/%d/delete", u1.User.ID),
				Form:           url.Values{"cancel": {"1"}},
				ExpectStatus:   303,
				ExpectRedirect: "/admin/users",
				Assert: func(t *testing.T, _ *Response) {
					// The task is not in the store anymore
					task := fmt.Sprintf("tasks:user.delete:%d", u1.User.ID)
					m := Store().Get(task)
					require.Empty(t, m)
				},
			},
		)
	})
}
