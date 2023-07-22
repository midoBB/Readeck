package signin

import (
	"errors"

	"github.com/doug-martin/goqu/v9"
	"github.com/readeck/readeck/internal/auth/users"
	"github.com/readeck/readeck/pkg/forms"
)

var errInvalidLogin = errors.New("Invalid user and/or password")

type tokenLoginForm struct {
	*forms.Form
}

func newTokenLoginForm() *tokenLoginForm {
	return &tokenLoginForm{forms.Must(
		forms.NewTextField("username", forms.Trim, forms.Required),
		forms.NewTextField("password", forms.Required),
		forms.NewTextField("application", forms.Required),
	)}
}

type loginForm struct {
	*forms.Form
}

func newLoginForm() *loginForm {
	return &loginForm{forms.Must(
		forms.NewTextField("username", forms.Trim, forms.Required),
		forms.NewTextField("password", forms.Required),
		forms.NewTextField("redirect", forms.Trim),
	)}
}

func (f *loginForm) checkUser() *users.User {
	user, err := users.Users.GetOne(goqu.C("username").Eq(f.Get("username").String()))
	if err != nil {
		f.AddErrors("", errInvalidLogin)
		return nil
	}
	if !user.CheckPassword(f.Get("password").String()) {
		f.AddErrors("", errInvalidLogin)
		return nil
	}

	return user
}
