// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package onboarding

import (
	"fmt"

	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/pkg/forms"
)

type onboardingForm struct {
	*forms.Form
}

func newOnboardingForm(tr forms.Translator) (f *onboardingForm) {
	f = &onboardingForm{forms.Must(
		forms.NewTextField("username", forms.Trim, forms.Required),
		forms.NewTextField("email", forms.Trim, forms.Optional(
			forms.IsEmail,
		)),
		forms.NewTextField("password", forms.Chain(
			forms.Required,
			users.IsValidPassword,
		)),
	)}
	f.SetLocale(tr)
	return
}

func (f *onboardingForm) createUser(language string) (*users.User, error) {
	u := &users.User{
		Username: f.Get("username").String(),
		Email:    f.Get("email").String(),
		Password: f.Get("password").String(),
		Group:    "admin",
		Settings: &users.UserSettings{
			Lang: language,
		},
	}

	if u.Email == "" {
		u.Email = fmt.Sprintf("%s@localhost", u.Username)
	}

	err := users.Users.Create(u)
	if err != nil {
		f.AddErrors("", forms.ErrUnexpected)
	}

	return u, err
}
