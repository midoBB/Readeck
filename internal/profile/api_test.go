package profile_test

import (
	"testing"

	. "github.com/readeck/readeck/internal/testing"
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
						"application":"tests"
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
					"seed": "<<PRESENCE>>",
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
							"value":null,
							"errors":null
						},
						"username":{
							"value":"",
							"errors":["cannot be blank"]
						}
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
						"value":"",
						"errors":null
					},
					"password":{
						"value":"  ",
						"errors":["password must be at least 8 character long"]
					}
				}
			}`,
		},
	)
}
