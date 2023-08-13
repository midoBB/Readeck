// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package sessions_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/readeck/readeck/internal/sessions"
)

func TestSession(t *testing.T) {
	hk := []byte("aaa0defe5d2839cbc46fc4f080cd7adc")
	bk := []byte("aaa0defe5d2839cbc46fc4f080cd7adc")

	t.Run("new session", func(t *testing.T) {
		handler := sessions.NewHandler("sid", hk, bk)

		r := httptest.NewRequest("GET", "/", nil)
		session, err := handler.New(r)
		assert.Nil(t, err)
		assert.True(t, session.IsNew)
	})

	t.Run("load session", func(t *testing.T) {
		handler := sessions.NewHandler("sid", hk, bk,
			sessions.MaxAge(60),
			sessions.Path("/test"),
		)

		r := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		session, err := handler.New(r)
		assert.Nil(t, err)
		assert.True(t, session.IsNew)

		session.Payload.User = 2
		session.AddFlash("info", "woot")
		err = session.Save(r, w)
		assert.Nil(t, err)
		assert.Len(t, w.Result().Cookies(), 1)

		cookie := w.Result().Cookies()[0]
		assert.Equal(t, "/test", cookie.Path)
		assert.True(t, cookie.HttpOnly)

		// Load the session
		r = httptest.NewRequest("GET", "/test", nil)
		r.AddCookie(cookie)
		session, err = handler.New(r)
		assert.Nil(t, err)
		assert.False(t, session.IsNew)
		assert.Equal(t, 2, session.Payload.User)
		assert.Len(t, session.Payload.Flashes, 1)
	})

	t.Run("flash", func(t *testing.T) {
		handler := sessions.NewHandler("sid", hk, bk,
			sessions.MaxAge(60),
			sessions.Path("/test"),
		)

		r := httptest.NewRequest("GET", "/", nil)
		session, err := handler.New(r)
		assert.Nil(t, err)
		session.AddFlash("info", "woot")

		assert.Len(t, session.Payload.Flashes, 1)
		flashes := session.Flashes()
		assert.Len(t, session.Payload.Flashes, 0)
		assert.Len(t, flashes, 1)

		assert.Equal(t, []sessions.FlashMessage{
			{"info", "woot"},
		}, flashes)
	})

	t.Run("ttl", func(t *testing.T) {
		handler := sessions.NewHandler("sid", hk, bk,
			sessions.MaxAge(1),
			sessions.Path("/test"),
		)

		r := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		session, err := handler.New(r)
		assert.Nil(t, err)
		assert.True(t, session.IsNew)

		err = session.Save(r, w)
		assert.Nil(t, err)
		assert.Len(t, w.Result().Cookies(), 1)

		cookie := w.Result().Cookies()[0]

		// Load the session
		r = httptest.NewRequest("GET", "/test", nil)
		r.AddCookie(cookie)
		session, err = handler.New(r)
		assert.Nil(t, err)
		assert.False(t, session.IsNew)

		// Cannot use the cookie after expiration, event if forged
		time.Sleep(2 * time.Second)
		r = httptest.NewRequest("GET", "/test", nil)
		r.AddCookie(cookie)
		session, err = handler.New(r)
		assert.True(t, session.IsNew)
		assert.NotNil(t, err)
		assert.Equal(t, "securecookie: expired timestamp", err.Error())
	})
}
