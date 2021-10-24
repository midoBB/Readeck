package signin

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/go-chi/chi/v5"
	"github.com/lithammer/shortuuid"

	"github.com/readeck/readeck/configs"
	"github.com/readeck/readeck/internal/auth/users"
	"github.com/readeck/readeck/internal/email"
	"github.com/readeck/readeck/internal/server"
	"github.com/readeck/readeck/pkg/forms"
)

var recoverCodes = map[string]int{}
var recoverDelay = time.Duration(2 * time.Hour)

type recoverForm struct {
	*forms.Form
}

func newRecoverForm() *recoverForm {
	return &recoverForm{forms.Must(
		forms.NewIntegerField("step", forms.Required),
		forms.NewTextField("email", forms.Trim),
		forms.NewTextField("password"),
	)}
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

func (h *authHandler) recover(w http.ResponseWriter, r *http.Request) {
	f := newRecoverForm()
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
		if err != nil && !errors.Is(err, users.ErrNotFound) {
			f.AddErrors("", forms.ErrUnexpected)
			return
		}

		mailTc := server.TC{
			"SiteURL":   h.srv.AbsoluteURL(r, "/"),
			"EmailAddr": f.Get("email").String(),
		}
		code := shortuuid.New()
		if user != nil {
			recoverCodes[code] = user.ID

			time.AfterFunc(recoverDelay, func() {
				delete(recoverCodes, code)
			})

			mailTc["RecoverLink"] = h.srv.AbsoluteURL(r, "/login/recover", code)
		}
		err = email.SendEmail(
			fmt.Sprintf("Readeck <%s>", configs.Config.Email.FromNoReply),
			f.Get("email").String(),
			"Password recovery",
			"recover.tmpl", mailTc,
		)
		if err != nil {
			h.srv.Log(r).WithError(err).Error("sending email")
			f.AddErrors("", forms.ErrUnexpected)
			return
		}

		f.Get("step").Set(1)
	}

	step2 := func() {
		var err error
		var user *users.User

		userID, ok := recoverCodes[recoverCode]
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

		delete(recoverCodes, recoverCode)
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
