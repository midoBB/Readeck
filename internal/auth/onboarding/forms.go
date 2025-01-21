// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package onboarding

import (
	"context"
	"fmt"

	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/pkg/forms"
)

type onboardingForm struct {
	*forms.Form
}

func newOnboardingForm(tr forms.Translator) *onboardingForm {
	return &onboardingForm{forms.Must(
		forms.WithTranslator(context.Background(), tr),
		forms.NewTextField("username", forms.Trim, forms.Required, users.IsValidUsername),
		forms.NewTextField("email", forms.Trim, forms.Skip, forms.IsEmail),
		forms.NewTextField("password", forms.Required, users.IsValidPassword),
	)}
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
