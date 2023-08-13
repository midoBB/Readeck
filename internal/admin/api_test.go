// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package admin_test

import (
	"fmt"
	"testing"

	. "github.com/readeck/readeck/internal/testing"
)

func TestAPI(t *testing.T) {
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
				JSON:         true,
				Target:       "/api/admin/users",
				ExpectStatus: 200,
				ExpectJSON: `[
					{
						"id": "<<PRESENCE>>",
						"href": "<<PRESENCE>>",
						"created": "<<PRESENCE>>",
						"updated": "<<PRESENCE>>",
						"username": "admin",
						"email": "admin@localhost",
						"group": "admin",
						"is_deleted": false
					},
					{
						"id": "<<PRESENCE>>",
						"href": "<<PRESENCE>>",
						"created": "<<PRESENCE>>",
						"updated": "<<PRESENCE>>",
						"username": "disabled",
						"email": "disabled@localhost",
						"group": "none",
						"is_deleted": false
					},
					{
						"id": "<<PRESENCE>>",
						"href": "<<PRESENCE>>",
						"created": "<<PRESENCE>>",
						"updated": "<<PRESENCE>>",
						"username": "staff",
						"email": "staff@localhost",
						"group": "staff",
						"is_deleted": false
					},
					{
						"id": "<<PRESENCE>>",
						"href": "<<PRESENCE>>",
						"created": "<<PRESENCE>>",
						"updated": "<<PRESENCE>>",
						"username": "test1",
						"email": "test1@localhost",
						"group": "user",
						"is_deleted": false
					},
					{
						"id": "<<PRESENCE>>",
						"href": "<<PRESENCE>>",
						"created": "<<PRESENCE>>",
						"updated": "<<PRESENCE>>",
						"username": "user",
						"email": "user@localhost",
						"group": "user",
						"is_deleted": false
					}
				]`,
			},
			RequestTest{
				JSON:         true,
				Target:       fmt.Sprintf("/api/admin/users/%d", u1.User.ID),
				ExpectStatus: 200,
				ExpectJSON: `{
					"id": "<<PRESENCE>>",
					"href": "<<PRESENCE>>",
					"created": "<<PRESENCE>>",
					"updated": "<<PRESENCE>>",
					"username": "test1",
					"email": "test1@localhost",
					"group": "user",
					"is_deleted": false,
					"settings": "<<PRESENCE>>"
				}`,
			},
			RequestTest{
				JSON:         true,
				Target:       "/api/admin/users/999999",
				ExpectStatus: 404,
				ExpectJSON:   `{"status":404,"message":"Not Found"}`,
			},
			RequestTest{
				Method:       "POST",
				Target:       "/api/admin/users",
				JSON:         map[string]string{},
				ExpectStatus: 422,
				ExpectJSON: `{
					"is_valid": false,
					"errors": null,
					"fields": {
						"email": {
							"is_bound": false,
							"is_null": true,
							"value": null,
							"errors": [
								"field is required"
							]
						},
						"group": {
							"is_bound": false,
							"is_null": false,
							"value": "user",
							"errors": ["field is required"]
						},
						"password": {
							"is_bound": false,
							"is_null": true,
							"value": null,
							"errors": [
								"field is required"
							]
						},
						"username": {
							"is_bound": false,
							"is_null": true,
							"value": null,
							"errors": [
								"field is required"
							]
						}
					}
				}`,
			},
			RequestTest{
				Method: "POST",
				Target: "/api/admin/users",
				JSON: map[string]string{
					"group": "foo",
				},
				ExpectStatus: 422,
				ExpectJSON: `{
					"is_valid": false,
					"errors": null,
					"fields": {
						"email": {
							"is_bound": false,
							"is_null": true,
							"value": null,
							"errors": [
								"field is required"
							]
						},
						"group": {
							"is_bound": true,
							"is_null": false,
							"value": "foo",
							"errors": ["must be one of none, user, staff, admin"]
						},
						"password": {
							"is_bound": false,
							"is_null": true,
							"value": null,
							"errors": [
								"field is required"
							]
						},
						"username": {
							"is_bound": false,
							"is_null": true,
							"value": null,
							"errors": [
								"field is required"
							]
						}
					}
				}`,
			},
			RequestTest{
				Method: "POST",
				Target: "/api/admin/users",
				JSON: map[string]string{
					"username": "user",
					"email":    "test2@localhost",
					"group":    "user",
					"password": "1234",
				},
				ExpectStatus: 422,
				ExpectJSON: `{
					"is_valid": false,
					"errors": null,
					"fields": {
						"email": {
							"is_bound": true,
							"is_null": false,
							"value": "test2@localhost",
							"errors": null
						},
						"group": {
							"is_bound": true,
							"is_null": false,
							"value": "user",
							"errors": null
						},
						"password": {
							"is_bound": true,
							"is_null": false,
							"value": "1234",
							"errors": null
						},
						"username": {
							"is_bound": true,
							"is_null": false,
							"value": "user",
							"errors": [
								"username is already in use"
							]
						}
					}
				}`,
			},
			RequestTest{
				Method: "POST",
				Target: "/api/admin/users",
				JSON: map[string]string{
					"username": "test2",
					"email":    "user@localhost",
					"group":    "user",
					"password": "1234",
				},
				ExpectStatus: 422,
				ExpectJSON: `{
					"is_valid": false,
					"errors": null,
					"fields": {
						"email": {
							"is_bound": true,
							"is_null": false,
							"value": "user@localhost",
							"errors": ["email address is already in use"]
						},
						"group": {
							"is_bound": true,
							"is_null": false,
							"value": "user",
							"errors": null
						},
						"password": {
							"is_bound": true,
							"is_null": false,
							"value": "1234",
							"errors": null
						},
						"username": {
							"is_bound": true,
							"is_null": false,
							"value": "test2",
							"errors": null
						}
					}
				}`,
			},
			RequestTest{
				Method: "POST",
				Target: "/api/admin/users",
				JSON: map[string]string{
					"username": "test2",
					"email":    "test2@localhost",
					"group":    "user",
					"password": "1234",
				},
				ExpectStatus:   201,
				ExpectJSON:     `{"status":201,"message":"User created"}`,
				ExpectRedirect: `/api/admin/users/\d+`,
			},
			RequestTest{
				Method:       "PATCH",
				Target:       fmt.Sprintf("/api/admin/users/%d", u1.User.ID),
				JSON:         map[string]string{},
				ExpectStatus: 200,
				ExpectJSON: `{
					"id": "<<PRESENCE>>"
				}`,
			},
			RequestTest{
				Method: "PATCH",
				Target: fmt.Sprintf("/api/admin/users/%d", u1.User.ID),
				JSON: map[string]string{
					"username": "test3",
					"email":    "test3@localhost",
					"group":    "user",
					"password": "2345",
				},
				ExpectStatus: 200,
				ExpectJSON: `{
					"id": "<<PRESENCE>>",
					"email": "test3@localhost",
					"group": "user",
					"password": "-",
					"updated": "<<PRESENCE>>",
					"username": "test3"
				}`,
			},
			RequestTest{
				Method:       "DELETE",
				Target:       fmt.Sprintf("/api/admin/users/%d", u1.User.ID),
				JSON:         true,
				ExpectStatus: 204,
			},
			RequestTest{
				Method:       "DELETE",
				Target:       fmt.Sprintf("/api/admin/users/%d", app.Users["admin"].User.ID),
				JSON:         true,
				ExpectStatus: 409,
				ExpectJSON: `{
					"status": 409,
					"message": "same user as authenticated"
				}`,
			},
		)
	})
}
