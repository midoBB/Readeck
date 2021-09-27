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
	"github.com/readeck/readeck/pkg/form"
)

var recoverCodes = map[string]int{}

type recoverForm struct {
	Step     int    `json:"step"`
	Email    string `json:"email"`
	Passowrd string `json:"password"`
}

var recoverDelay = time.Duration(2 * time.Hour)

func (rf *recoverForm) Validate(f *form.Form) {
	switch rf.Step {
	case 0, 1:
		f.Get("email").Validate(
			form.IsRequiredOrNull, form.IsValidEmail,
		)
	case 2, 3:
		f.Get("password").Validate(form.IsRequired, users.IsValidPassword)
	}
}

func (h *authHandler) recover(w http.ResponseWriter, r *http.Request) {
	rf := &recoverForm{}
	f := form.NewForm(rf)

	tc := server.TC{
		"Form": f,
	}

	recoverCode := chi.URLParam(r, "code")

	step0 := func() {
		if !f.IsValid() {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		user, err := users.Users.GetOne(goqu.C("email").Eq(rf.Email))
		if err != nil && !errors.Is(err, users.ErrNotFound) {
			f.Errors().Add(errors.New("An error occurred"))
			return
		}

		mailTc := server.TC{
			"SiteURL":   h.srv.AbsoluteURL(r, "/"),
			"EmailAddr": rf.Email,
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
			rf.Email,
			"Password recovery",
			"recover.tmpl", mailTc,
		)
		if err != nil {
			h.srv.Log(r).WithError(err).Error("sending email")
			f.Errors().Add(errors.New("An error occurred while sending the email"))
			return
		}

		rf.Step = 1
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
				f.Errors().Add(errors.New("An error occurred"))
			}
		}()

		if err = user.SetPassword(rf.Passowrd); err != nil {
			return
		}
		user.SetSeed()
		if err = user.Save(); err != nil {
			return
		}

		delete(recoverCodes, recoverCode)
		rf.Step = 3
	}

	switch r.Method {
	case http.MethodGet:
		if recoverCode != "" {
			rf.Step = 2
			step2()
		}
	case http.MethodPost:
		form.Bind(f, r)
		switch rf.Step {
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
