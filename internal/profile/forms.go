// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package profile

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"

	"codeberg.org/readeck/readeck/internal/auth/credentials"
	"codeberg.org/readeck/readeck/internal/auth/tokens"
	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/internal/db/types"
	"codeberg.org/readeck/readeck/locales"
	"codeberg.org/readeck/readeck/pkg/forms"
)

type (
	ctxUserFormKey struct{}
)

// profileForm is the form used by the profile update routes.
type profileForm struct {
	*forms.Form
}

var (
	errInvalidUserOrEmail = forms.Gettext("invalid username and/or email")
	errInvalidPassword    = forms.Gettext("invalid password")
)

// newProfileForm returns a ProfileForm instance.
func newProfileForm(tr forms.Translator) (f *profileForm) {
	f = &profileForm{
		forms.Must(
			forms.NewTextField("username",
				forms.Trim, forms.RequiredOrNil, users.IsValidUsername),
			forms.NewTextField("email",
				forms.Trim, forms.RequiredOrNil, forms.IsEmail),
			forms.NewChoiceField("settings_lang",
				locales.Available(),
				forms.Trim, forms.RequiredOrNil,
			),
			forms.NewTextField("settings_reader_font",
				forms.Trim, forms.RequiredOrNil,
			),
			forms.NewIntegerField("settings_reader_font_size",
				forms.RequiredOrNil, forms.Gte(1), forms.Lte(6),
			),
			forms.NewIntegerField("settings_reader_line_height",
				forms.RequiredOrNil, forms.Gte(1), forms.Lte(6),
			),
		),
	}
	f.SetLocale(tr)

	return
}

// setUser sets the form's values from a user instance.
func (f *profileForm) setUser(u *users.User) {
	ctx := context.WithValue(f.Context(), ctxUserFormKey{}, u)
	f.SetContext(ctx)

	f.Get("username").Set(u.Username)
	f.Get("email").Set(u.Email)
	f.Get("settings_lang").Set(u.Settings.Lang)
}

// Validate performs extra validation.
func (f *profileForm) Validate() {
	u, _ := f.Context().Value(ctxUserFormKey{}).(*users.User)

	for _, field := range f.Fields() {
		if !field.IsBound() || field.IsNil() {
			continue
		}

		if u == nil {
			continue
		}

		switch field.Name() {
		// check if username and/or email is not already in use.
		case "username", "email":
			c, err := users.Users.Query().Where(
				goqu.C(field.Name()).Eq(field.String()),
				goqu.C("id").Neq(u.ID),
			).Count()
			if err != nil {
				f.AddErrors("", forms.ErrUnexpected)
				return
			}
			if c > 0 {
				f.AddErrors("", errInvalidUserOrEmail)
				return
			}
		}
	}
}

// updateUser updates the given user using the form's values.
func (f *profileForm) updateUser(u *users.User) (res map[string]interface{}, err error) {
	if !f.IsBound() {
		err = errors.New("form is not bound")
		return
	}

	resetSeed := false
	res = make(map[string]interface{})
	for _, field := range f.Fields() {
		if !field.IsBound() || field.IsNil() {
			continue
		}

		switch n := field.Name(); {
		case strings.HasPrefix(n, "settings_reader_"):
			name := strings.TrimPrefix(n, "settings_reader_")
			switch name {
			case "font":
				u.Settings.ReaderSettings.Font = field.String()
			case "font_size":
				u.Settings.ReaderSettings.FontSize = field.Value().(int)
			case "line_height":
				u.Settings.ReaderSettings.LineHeight = field.Value().(int)
			}
			res["settings"] = u.Settings
		case n == "settings_lang":
			u.Settings.Lang = field.String()
			res["settings"] = u.Settings
		default:
			if n == "email" || n == "username" {
				resetSeed = true
			}
			res[field.Name()] = field.Value()
		}

	}

	if len(res) > 0 {
		res["updated"] = time.Now()
		if resetSeed {
			res["seed"] = u.SetSeed()
		}
		if err = u.Update(res); err != nil {
			f.AddErrors("", forms.ErrUnexpected)
			return
		}

	}
	res["id"] = u.ID
	delete(res, "seed")
	return
}

// passwordForm is a form to update a user's password.
type passwordForm struct {
	*forms.Form
}

// newPasswordForm returns a PasswordForm instance.
func newPasswordForm() *passwordForm {
	f, err := forms.New(
		forms.NewTextField("current"),
		forms.NewTextField("password",
			forms.Required, users.IsValidPassword),
	)
	if err != nil {
		panic(err)
	}

	return &passwordForm{f}
}

// setUser adds a user to the wrapping form's context.
func (f *passwordForm) setUser(u *users.User) {
	ctx := context.WithValue(f.Context(), ctxUserFormKey{}, u)
	f.SetContext(ctx)
}

// Validate performs extra validation steps.
func (f *passwordForm) Validate() {
	// If a user was passed in context, then "current"
	// is mandatory and must match the current user
	// password.
	u, ok := f.Context().Value(ctxUserFormKey{}).(*users.User)
	if !ok {
		return
	}

	if errs := forms.ValidateField(f.Get("current"), forms.Required); len(errs) > 0 {
		f.AddErrors("current", errs...)
	}
	if !f.IsValid() {
		return
	}
	if !u.CheckPassword(f.Get("current").String()) {
		f.AddErrors("current", errInvalidPassword)
	}
}

// updatePassword performs the user's password update.
func (f *passwordForm) updatePassword(u *users.User) (err error) {
	defer func() {
		if err != nil {
			f.AddErrors("", forms.ErrUnexpected)
		}
	}()

	if err = u.SetPassword(f.Get("password").String()); err != nil {
		return
	}
	err = u.Update(map[string]interface{}{"seed": u.SetSeed()})
	return
}

// deleteCredentialForm is the form used for credential deletion.
type deleteCredentialForm struct {
	*forms.Form
}

// newDeleteTokenForm returns a deleteForm instance.
func newDeleteCredentialForm(tr forms.Translator) (f *deleteCredentialForm) {
	f = &deleteCredentialForm{forms.Must(
		forms.NewBooleanField("cancel"),
		forms.NewTextField("_to"),
	)}
	f.SetLocale(tr)
	return
}

// trigger launch the token deletion or cancel task.
func (f *deleteCredentialForm) trigger(c *credentials.Credential) error {
	if !f.Get("cancel").IsNil() && f.Get("cancel").Value().(bool) {
		return deleteCredentialTask.Cancel(c.ID)
	}

	return deleteCredentialTask.Run(c.ID, c.ID)
}

type credentialForm struct {
	*forms.Form
}

// newCredentialForm returns an credentialForm instance.
func newCredentialForm(tr forms.Translator, user *users.User) (f *credentialForm) {
	f = &credentialForm{forms.Must(
		forms.NewBooleanField("is_enabled", forms.RequiredOrNil),
		forms.NewTextField("name", forms.Required, forms.Trim),
		users.NewRolesField(user),
	)}
	f.SetLocale(tr)
	return
}

// setCredential set the token's values from an existing token.
func (f *credentialForm) setCredential(p *credentials.Credential) {
	f.Get("is_enabled").Set(p.IsEnabled)
	f.Get("name").Set(p.Name)

	roles := make([]string, len(p.Roles))
	copy(roles, p.Roles)
	f.Get("roles").Set(roles)
}

// updateCredential performs the credential update.
func (f *credentialForm) updateCredential(p *credentials.Credential) error {
	for _, field := range f.Fields() {
		if !field.IsBound() {
			continue
		}
		switch field.Name() {
		case "is_enabled":
			p.IsEnabled = field.Value().(bool)
		case "name":
			p.Name = field.String()
		case "roles":
			if field.Value() != nil {
				p.Roles = field.Value().(types.Strings)
			} else {
				p.Roles = nil
			}
		}
	}

	if err := p.Save(); err != nil {
		f.AddErrors("", forms.ErrUnexpected)
		return err
	}
	return nil
}

// deleteTokenForm is the form used for token deletion.
type deleteTokenForm struct {
	*forms.Form
}

// newDeleteTokenForm returns a deleteForm instance.
func newDeleteTokenForm(tr forms.Translator) (f *deleteTokenForm) {
	f = &deleteTokenForm{forms.Must(
		forms.NewBooleanField("cancel"),
		forms.NewTextField("_to"),
	)}
	f.SetLocale(tr)
	return
}

// trigger launch the token deletion or cancel task.
func (f *deleteTokenForm) trigger(t *tokens.Token) error {
	if !f.Get("cancel").IsNil() && f.Get("cancel").Value().(bool) {
		return deleteTokenTask.Cancel(t.ID)
	}

	return deleteTokenTask.Run(t.ID, t.ID)
}

// tokenForm is the form used for token update.
type tokenForm struct {
	*forms.Form
}

// tokenForm returns a tokenForm instance.
func newTokenForm(tr forms.Translator, user *users.User) (f *tokenForm) {
	f = &tokenForm{forms.Must(
		forms.NewBooleanField("is_enabled", forms.RequiredOrNil),
		forms.NewDatetimeField("expires"),
		users.NewRolesField(user),
	)}
	f.SetLocale(tr)
	return
}

// setToken set the token's values from an existing token.
func (f *tokenForm) setToken(t *tokens.Token) {
	f.Get("is_enabled").Set(t.IsEnabled)
	f.Get("expires").Set(t.Expires)

	roles := make([]string, len(t.Roles))
	copy(roles, t.Roles)
	f.Get("roles").Set(roles)
}

// updateToken performs the token update.
func (f *tokenForm) updateToken(t *tokens.Token) error {
	for _, field := range f.Fields() {
		if !field.IsBound() {
			continue
		}
		switch field.Name() {
		case "is_enabled":
			t.IsEnabled = field.Value().(bool)
		case "expires":
			if field.Value() == nil {
				t.Expires = nil
				continue
			}
			v := field.Value().(time.Time)
			t.Expires = &v
		case "roles":
			if field.Value() != nil {
				t.Roles = field.Value().(types.Strings)
			} else {
				t.Roles = nil
			}
		}

	}

	if err := t.Save(); err != nil {
		f.AddErrors("", forms.ErrUnexpected)
		return err
	}
	return nil
}
