// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package signin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/go-chi/chi/v5"
	"github.com/lithammer/shortuuid/v4"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/internal/bus"
	"codeberg.org/readeck/readeck/internal/email"
	"codeberg.org/readeck/readeck/internal/server"
	"codeberg.org/readeck/readeck/pkg/forms"
)

type recoverForm struct {
	*forms.Form
	ttl    time.Duration
	prefix string
}

func newRecoverForm(tr forms.Translator) (f *recoverForm) {
	f = &recoverForm{
		Form: forms.Must(
			forms.NewIntegerField("step", forms.Required),
			forms.NewTextField("email", forms.Trim),
			forms.NewTextField("password"),
		),
		ttl:    time.Duration(2 * time.Hour),
		prefix: "recover_code",
	}
	f.SetLocale(tr)
	return
}

func (f *recoverForm) Validate() {
	if !f.IsValid() {
		return
	}

	switch f.Get("step").Value().(int) {
	case 0, 1:
		f.AddErrors("email", forms.ValidateField(f.Get("email"), forms.Required)...)
	case 2, 3:
		f.AddErrors("password", forms.ValidateField(f.Get("password"), forms.Required, users.IsValidPassword)...)
	default:
		f.AddErrors("", errors.New("invalid step"))
	}
}

func (f *recoverForm) saveCode(code string, userID int) error {
	return bus.Store().Set(fmt.Sprintf("%s_%s", f.prefix, code), fmt.Sprint(userID), f.ttl)
}

func (f *recoverForm) getCode(code string) (int, bool) {
	v := bus.Store().Get(fmt.Sprintf("%s_%s", f.prefix, code))
	if v == "" {
		return 0, false
	}
	userID, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return userID, true
}

func (f *recoverForm) delCode(code string) error {
	return bus.Store().Del(fmt.Sprintf("%s_%s", f.prefix, code))
}

func (h *authHandler) recover(w http.ResponseWriter, r *http.Request) {
	f := newRecoverForm(h.srv.Locale(r))
	f.Get("step").Set(0)

	tc := server.TC{
		"Form": f,
	}

	recoverCode := chi.URLParam(r, "code")

	step0 := func() {
		if !f.IsValid() {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		user, err := users.Users.GetOne(goqu.C("email").Eq(f.Get("email").String()))

		defer func() {
			if err != nil {
				h.srv.Log(r).WithError(err).Error("recover step 0")
				f.AddErrors("", forms.ErrUnexpected)
			}
		}()

		if err != nil && !errors.Is(err, users.ErrNotFound) {
			return
		}

		mailTc := server.TC{
			"SiteURL":   h.srv.AbsoluteURL(r, "/"),
			"EmailAddr": f.Get("email").String(),
		}
		code := shortuuid.New()
		if user != nil {
			if err = f.saveCode(code, user.ID); err != nil {
				return
			}

			mailTc["RecoverLink"] = h.srv.AbsoluteURL(r, "/login/recover", code)
		}
		err = email.SendEmail(
			fmt.Sprintf("Readeck <%s>", configs.Config.Email.FromNoReply),
			f.Get("email").String(),
			"Password recovery",
			"recover.tmpl", mailTc,
		)
		if err != nil {
			return
		}

		f.Get("step").Set(1)
	}

	step2 := func() {
		var err error
		var user *users.User

		userID, ok := f.getCode(recoverCode)
		if !ok {
			tc["Error"] = "Invalid recovery code"
			return
		}
		user, err = users.Users.GetOne(goqu.C("id").Eq(userID))
		if err != nil {
			tc["Error"] = "Invalid recovery code"
			h.srv.Log(r).WithError(err).Error("get user")
			return
		}

		if r.Method == http.MethodGet {
			return
		}

		if !f.IsValid() {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		defer func() {
			if err != nil {
				h.srv.Log(r).WithError(err).Error("password update")
				f.AddErrors("", forms.ErrUnexpected)
			}
		}()

		if err = user.SetPassword(f.Get("password").String()); err != nil {
			return
		}
		user.SetSeed()
		if err = user.Save(); err != nil {
			return
		}

		if err = f.delCode(recoverCode); err != nil {
			return
		}
		f.Get("step").Set(3)
	}

	switch r.Method {
	case http.MethodGet:
		if recoverCode != "" {
			f.Get("step").Set(2)
			step2()
		}
	case http.MethodPost:
		forms.Bind(f, r)
		switch f.Get("step").Value() {
		case 0:
			step0()
		case 1:
			// Step 1 is a template only step
			if recoverCode == "" || !f.IsValid() {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		case 2:
			step2()
		case 3:
			// Step 3 is a template only step
			if recoverCode == "" || !f.IsValid() {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}

	}

	h.srv.RenderTemplate(w, r, http.StatusOK, "/auth/recover", tc)
}
