// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package onboarding

import (
	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/pkg/forms"
)

type onboardingForm struct {
	*forms.Form
}

func newOnboardingForm() *onboardingForm {
	return &onboardingForm{forms.Must(
		forms.NewTextField("email", forms.Trim, forms.Chain(
			forms.Required,
			forms.IsEmail,
		)),
		forms.NewTextField("username", forms.Trim, forms.Required),
		forms.NewTextField("password", forms.Chain(
			forms.Required,
			users.IsValidPassword,
		)),
	)}
}

func (f *onboardingForm) createUser() (*users.User, error) {
	u := &users.User{
		Username: f.Get("username").String(),
		Email:    f.Get("email").String(),
		Password: f.Get("password").String(),
		Group:    "admin",
	}

	err := users.Users.Create(u)
	if err != nil {
		f.AddErrors("", forms.ErrUnexpected)
	}

	return u, err
}
