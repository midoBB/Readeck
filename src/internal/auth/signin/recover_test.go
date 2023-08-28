// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package signin_test

import (
	"fmt"
	"net/url"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	. "codeberg.org/readeck/readeck/internal/testing"
)

func TestRecover(t *testing.T) {
	app := NewTestApp(t)
	defer app.Close(t)

	client := NewClient(t, app)

	t.Run("recover views", func(t *testing.T) {
		token := ""
		RunRequestSequence(t, client, "",
			RequestTest{
				Target:       "/login/recover",
				ExpectStatus: 200,
			},
			RequestTest{
				Method: "POST",
				Target: "/login/recover",
				Form: url.Values{
					"step":  {"0"},
					"email": {"user@localhost"},
				},
				ExpectStatus: 200,
				Assert: func(t *testing.T, _ *Response) {
					assert.Contains(t, app.LastEmail, "login/recover/")
					rx := regexp.MustCompile(fmt.Sprintf(
						"%s/login/recover/(.+)\r\n",
						regexp.QuoteMeta("http://"+client.URL.Host),
					))
					m := rx.FindStringSubmatch(app.LastEmail)
					if len(m) < 2 {
						t.Fatal("could not find recovery link in last email")
					}
					token = m[1]
				},
			},
		)
		RunRequestSequence(t, client, "",
			RequestTest{
				Target:       "/login/recover/" + token,
				ExpectStatus: 200,
			},
			RequestTest{
				Method: "POST",
				Target: "/login/recover/" + token,
				Form: url.Values{
					"step":     {"2"},
					"password": {"09876543"},
				},
				ExpectStatus: 200,
			},
			RequestTest{Target: "/login"},
			RequestTest{
				Method: "POST",
				Target: "/login",
				Form: url.Values{
					"username": {"user"},
					"password": {"09876543"},
				},
				ExpectStatus:   303,
				ExpectRedirect: "/",
			},
		)
	})

	t.Run("recover no user", func(t *testing.T) {
		RunRequestSequence(t, client, "",
			RequestTest{
				Target:       "/login/recover",
				ExpectStatus: 200,
			},
			RequestTest{
				Method: "POST",
				Target: "/login/recover",
				Form: url.Values{
					"step":  {"0"},
					"email": {"nope@localhost"},
				},
				ExpectStatus: 200,
				Assert: func(t *testing.T, _ *Response) {
					assert.Contains(
						t, app.LastEmail,
						"However, this email address is not associated with any account",
					)
				},
			},
		)
	})

	t.Run("recover steps", func(t *testing.T) {
		RunRequestSequence(t, client, "",
			RequestTest{
				Target:         "/login/recover/abcdefghijkl",
				ExpectStatus:   200,
				ExpectContains: "Invalid recovery code",
			},
			RequestTest{
				Method:       "POST",
				Target:       "/login/recover/abcdefghijkl",
				Form:         url.Values{"password": {"09876543"}},
				ExpectStatus: 422,
			},
			RequestTest{Target: "/login"},
			RequestTest{
				Method: "POST",
				Target: "/login/recover/abcdefghijkl",
				Form: url.Values{
					"step":     {"2"},
					"password": {"09876543"},
				},
				ExpectStatus:   200,
				ExpectContains: "Invalid recovery code",
			},
		)
	})
}
