// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package profile_test

import (
	"fmt"
	"testing"

	. "codeberg.org/readeck/readeck/internal/testing" //revive:disable:dot-imports
)

func TestAPI(t *testing.T) {
	app := NewTestApp(t)
	defer func() {
		app.Close(t)
	}()

	client := NewClient(t, app)

	RunRequestSequence(t, client, "user",
		RequestTest{
			JSON:         true,
			Target:       "/api/profile",
			ExpectStatus: 200,
			ExpectJSON: `{
					"provider":{
						"name":"bearer token",
						"application":"tests",
						"id":"<<PRESENCE>>",
						"roles":["user"],
						"permissions":"<<PRESENCE>>"
					},
					"user":{
						"username":"user",
						"email":"user@localhost",
						"created":"<<PRESENCE>>",
						"updated":"<<PRESENCE>>",
						"settings": "<<PRESENCE>>"
					}
				}`,
		},
		RequestTest{
			Method:       "PATCH",
			Target:       "/api/profile",
			JSON:         map[string]interface{}{},
			ExpectStatus: 200,
			ExpectJSON: `{
					"id": {{ .Users.user.User.ID }}
				}`,
		},
		RequestTest{
			Method: "PATCH",
			Target: "/api/profile",
			JSON: map[string]interface{}{
				"username": " newuser ",
				"email":    " newuser@localhost ",
			},
			ExpectStatus: 200,
			ExpectJSON: `{
					"id": {{ .Users.user.User.ID }},
					"email": "newuser@localhost",
					"updated": "<<PRESENCE>>",
					"username":"newuser"
				}`,
		},
		RequestTest{
			Method: "PATCH",
			Target: "/api/profile",
			JSON: map[string]interface{}{
				"username": " ",
			},
			ExpectStatus: 422,
			ExpectJSON: `{
					"is_valid":false,
					"errors":null,
					"fields":{
						"email":{
							"is_null": false,
							"is_bound": false,
							"value": "<<PRESENCE>>",
							"errors":null
						},
						"username":{
							"is_null": false,
							"is_bound": true,
							"value":"",
							"errors":[
								"field is required",
        						"must contain English letters, digits, \"_\" and \"-\" only"
							]
						},
						"settings_lang": "<<PRESENCE>>",
						"settings_reader_font": "<<PRESENCE>>",
						"settings_reader_font_size": "<<PRESENCE>>",
						"settings_reader_line_height": "<<PRESENCE>>"
					}
				}`,
		},
		RequestTest{
			Method: "PATCH",
			Target: "/api/profile",
			JSON: map[string]interface{}{
				"username": "user@localhost",
				"email":    "user",
			},
			ExpectStatus: 422,
			ExpectJSON: `{
					"is_valid":false,
					"errors":null,
					"fields":{
						"email":{
							"is_null": false,
							"is_bound": true,
							"value": "user",
							"errors":[
								"not a valid email address"
							]
						},
						"username":{
							"is_null": false,
							"is_bound": true,
							"value":"user@localhost",
							"errors":[
        						"must contain English letters, digits, \"_\" and \"-\" only"
							]
						},
						"settings_lang": "<<PRESENCE>>",
						"settings_reader_font": "<<PRESENCE>>",
						"settings_reader_font_size": "<<PRESENCE>>",
						"settings_reader_line_height": "<<PRESENCE>>"
					}
				}`,
		},

		RequestTest{
			Method: "PUT",
			Target: "/api/profile/password",
			JSON: map[string]interface{}{
				"password": "newpassword",
			},
			ExpectStatus: 200,
		},
		RequestTest{
			Method: "PUT",
			Target: "/api/profile/password",
			JSON: map[string]interface{}{
				"password": "  ",
			},
			ExpectStatus: 422,
			ExpectJSON: `{
				"is_valid":false,
				"errors":null,
				"fields":{
					"current":{
						"is_null": true,
						"is_bound": false,
						"value":null,
						"errors":null
					},
					"password":{
						"is_null": false,
						"is_bound": true,
						"value":"  ",
						"errors":["password must be at least 8 character long"]
					}
				}
			}`,
		},
	)
}

func TestAPIDeleteToken(t *testing.T) {
	app := NewTestApp(t)
	defer func() {
		app.Close(t)
	}()

	client := NewClient(t, app)

	u1, err := NewTestUser("test1", "test1@localhost", "test1", "user")
	if err != nil {
		t.Fatal(err)
	}

	app.Users[u1.User.Username] = u1

	RunRequestSequence(t, client, "user",
		RequestTest{
			JSON:         true,
			Target:       fmt.Sprintf("/api/profile/tokens/%s", u1.Token.UID),
			Method:       "DELETE",
			ExpectStatus: 404,
		},
	)

	RunRequestSequence(t, client, u1.User.Username,
		RequestTest{
			JSON:         true,
			Target:       "/api/profile",
			ExpectStatus: 200,
		},
		RequestTest{
			JSON:         true,
			Target:       fmt.Sprintf("/api/profile/tokens/%s", u1.Token.UID),
			Method:       "DELETE",
			ExpectStatus: 204,
		},
		RequestTest{
			JSON:         true,
			Target:       "/api/profile",
			ExpectStatus: 401,
		},
	)
}
