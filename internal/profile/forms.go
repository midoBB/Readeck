package profile

import (
	"context"
	"errors"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/readeck/readeck/internal/auth/tokens"
	"github.com/readeck/readeck/internal/auth/users"
	"github.com/readeck/readeck/pkg/forms"
)

type (
	ctxUserFormKey struct{}
)

// profileForm is the form used by the profile update routes.
type profileForm struct {
	*forms.Form
}

// newProfileForm returns a ProfileForm instance.
func newProfileForm() *profileForm {
	f, err := forms.New(
		forms.NewTextField("username",
			forms.Trim, forms.RequiredOrNil, users.IsValidUsername),
		forms.NewTextField("email",
			forms.Trim, forms.RequiredOrNil, forms.IsEmail),
	)

	if err != nil {
		panic(err)
	}
	return &profileForm{f}
}

// setUser sets the form's values from a user instance.
func (f *profileForm) setUser(u *users.User) {
	ctx := context.WithValue(f.Context(), ctxUserFormKey{}, u)
	f.SetContext(ctx)

	f.Get("username").Set(u.Username)
	f.Get("email").Set(u.Email)
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
				f.AddErrors("", errors.New("invalid username and/or email"))
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

	res = make(map[string]interface{})
	for _, field := range f.Fields() {
		if !field.IsBound() || field.IsNil() {
			continue
		}

		res[field.Name()] = field.Value()
	}

	if len(res) > 0 {
		res["updated"] = time.Now()
		res["seed"] = u.SetSeed()
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
		f.AddErrors("current", errors.New("Invalid password"))
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

// deleteTokenForm is the form used for token deletion.
type deleteTokenForm struct {
	*forms.Form
}

// newDeleteTokenForm returns a deleteForm instance.
func newDeleteTokenForm() *deleteTokenForm {
	return &deleteTokenForm{forms.Must(
		forms.NewBooleanField("cancel"),
	)}
}

// trigger launch the token deletion or cancel task.
func (f *deleteTokenForm) trigger(t *tokens.Token) {
	if !f.Get("cancel").IsNil() && f.Get("cancel").Value().(bool) {
		deleteTokenTask.Cancel(t.ID)
		return
	}

	deleteTokenTask.Run(t.ID, t.ID)
}

// tokenForm is the form used for token update.
type tokenForm struct {
	*forms.Form
}

// tokenForm returns a tokenForm instance.
func newTokenForm() *tokenForm {
	return &tokenForm{forms.Must(
		forms.NewBooleanField("is_enabled", forms.RequiredOrNil),
		forms.NewDatetimeField("expires"),
	)}
}

// setToken set the token's values from an existing token.
func (f *tokenForm) setToken(t *tokens.Token) {
	f.Get("is_enabled").Set(t.IsEnabled)
	f.Get("expires").Set(t.Expires)
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
		}
	}

	if err := t.Save(); err != nil {
		f.AddErrors("", forms.ErrUnexpected)
		return err
	}
	return nil
}
