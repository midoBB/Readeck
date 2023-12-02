// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package signin

import (
	"errors"
	"strings"

	"github.com/doug-martin/goqu/v9"

	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/pkg/forms"
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
		users.NewRolesField(nil),
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

func checkUser(f forms.Binder) *users.User {
	col := goqu.C("username")
	if strings.Contains(f.Get("username").String(), "@") {
		// A username cannot contain a "@" so if we have one here,
		// we can check on the email instead of the username.
		col = goqu.C("email")
	}

	user, err := users.Users.GetOne(col.Eq(f.Get("username").String()))
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
