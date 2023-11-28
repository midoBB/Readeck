// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package signin_test

import (
	"testing"

	. "codeberg.org/readeck/readeck/internal/testing" //revive:disable:dot-imports
)

func TestAPI(t *testing.T) {
	app := NewTestApp(t)
	defer app.Close(t)

	client := NewClient(t, app)

	RunRequestSequence(t, client, "",
		RequestTest{
			Method:       "POST",
			Target:       "/api/auth",
			JSON:         map[string]string{},
			ExpectStatus: 400,
		},
		RequestTest{
			Method: "POST",
			Target: "/api/auth",
			JSON: map[string]string{
				"application": "test",
				"username":    "admin",
				"password":    "nope",
			},
			ExpectStatus: 403,
			ExpectJSON: `{
				"status":403,
				"message":"Invalid user and/or password"
			}`,
		},
		RequestTest{
			Method: "POST",
			Target: "/api/auth",
			JSON: map[string]string{
				"application": "test",
				"username":    "admin",
				"password":    "admin",
			},
			ExpectStatus: 201,
			ExpectJSON: `{
					"id": "<<PRESENCE>>",
					"token": "<<PRESENCE>>"
			}`,
		},
	)
}
