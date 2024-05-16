// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package sessions provides a cookie based session manager.
// It's heavily based on gorilla session but with a structured session
// payload that can be serialized to json.
package sessions

import (
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
)

// Payload contains session values.
type Payload struct {
	Seed        int            `json:"s"`
	User        int            `json:"u"`
	Flashes     []FlashMessage `json:"f"`
	Preferences Preferences    `json:"p"`
}

// FlashMessage is a message stored in the session.
type FlashMessage struct {
	Type    string `json:"t"`
	Message string `json:"m"`
}

// Preferences contains the user session preferences.
type Preferences struct {
	LastUpdate          time.Time `json:"u"`
	BookmarkListDisplay string    `json:"bld"`
}

// Session is a unique session.
type Session struct {
	Payload *Payload
	IsNew   bool
	MaxAge  int
	handler *Handler
}

// Handler is the session handler with its global options.
type Handler struct {
	sc *securecookie.SecureCookie

	// Cookie options
	name   string
	path   string
	maxAge int
}

// Path sets the session handler's path.
func Path(val string) func(s *Handler) {
	return func(s *Handler) {
		s.path = val
	}
}

// MaxAge sets the session handler's max age.
func MaxAge(val int) func(s *Handler) {
	return func(s *Handler) {
		s.maxAge = val
		s.sc.MaxAge(val)
	}
}

// NewHandler creates a session handler.
func NewHandler(name string, hashKey, blockKey []byte, options ...func(s *Handler)) *Handler {
	s := &Handler{
		sc:   securecookie.New(hashKey, blockKey),
		name: name,
	}
	s.sc.SetSerializer(securecookie.JSONEncoder{})

	for _, f := range options {
		f(s)
	}

	return s
}

// New creates or retrieves a session.
func (s *Handler) New(r *http.Request) (*Session, error) {
	session := &Session{
		Payload: new(Payload),
		IsNew:   true,
		MaxAge:  s.maxAge,
		handler: s,
	}
	var err error

	if c, _err := r.Cookie(s.name); _err == nil {
		err = s.sc.Decode(s.name, c.Value, session.Payload)
		if err == nil {
			session.IsNew = false
		}
	}

	return session, err
}

// Save encodes the session's data and encloses it in a cookie that's written
// on the HTTP response.
func (s *Handler) Save(r *http.Request, w http.ResponseWriter, session *Session) error {
	encoded, err := s.sc.Encode(s.name, session.Payload)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:     s.name,
		Value:    encoded,
		Path:     s.path,
		MaxAge:   session.MaxAge,
		Secure:   r.URL.Scheme == "https",
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	}

	if session.MaxAge > 0 {
		d := time.Duration(session.MaxAge) * time.Second
		cookie.Expires = time.Now().Add(d)
	} else if session.MaxAge < 0 {
		cookie.Expires = time.Unix(1, 0)
	}

	http.SetCookie(w, cookie)
	return nil
}

// Save sends the session's cookie to the HTTP response.
func (s *Session) Save(r *http.Request, w http.ResponseWriter) error {
	return s.handler.Save(r, w, s)
}

// AddFlash add a new flash message to the session.
func (s *Session) AddFlash(typ, msg string) {
	s.Payload.Flashes = append(s.Payload.Flashes, FlashMessage{typ, msg})
}

// Flashes retrieves the flash messages from the session
// and flushes them. The session is not saved, it's up to the code (middleware)
// that calls this function to save the session.
func (s *Session) Flashes() []FlashMessage {
	defer func() {
		s.Payload.Flashes = make([]FlashMessage, 0)
	}()
	return s.Payload.Flashes
}
