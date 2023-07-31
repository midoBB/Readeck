package profile

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"

	"github.com/readeck/readeck/internal/acls"
	"github.com/readeck/readeck/internal/auth/tokens"
	"github.com/readeck/readeck/internal/auth/users"
	"github.com/readeck/readeck/internal/db"
	"github.com/readeck/readeck/pkg/forms"
)

type (
	ctxUserFormKey struct{}
)

// profileForm is the form used by the profile update routes.
type profileForm struct {
	*forms.Form
}

var availableScopes = [][2]string{
	{"scoped_bookmarks_r", "Bookmarks : Read Only"},
	{"scoped_bookmarks_w", "Bookmarks : Write Only"},
	{"scoped_admin_r", "Admin : Read Only"},
	{"scoped_admin_w", "Admin : Write Only"},
}

// newProfileForm returns a ProfileForm instance.
func newProfileForm() *profileForm {
	f, err := forms.New(
		forms.NewTextField("username",
			forms.Trim, forms.RequiredOrNil, users.IsValidUsername),
		forms.NewTextField("email",
			forms.Trim, forms.RequiredOrNil, forms.IsEmail),
		forms.NewTextField("settings_reader_font",
			forms.Trim, forms.RequiredOrNil,
		),
		forms.NewIntegerField("settings_reader_font_size",
			forms.RequiredOrNil, forms.Gte(1), forms.Lte(6),
		),
		forms.NewIntegerField("settings_reader_line_height",
			forms.RequiredOrNil, forms.Gte(1), forms.Lte(6),
		),
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
		forms.NewTextField("_to"),
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
func newTokenForm(user *users.User) *tokenForm {
	return &tokenForm{forms.Must(
		forms.NewBooleanField("is_enabled", forms.RequiredOrNil),
		forms.NewDatetimeField("expires"),
		newRolesField(user),
	)}
}

func newRolesField(user *users.User) forms.Field {
	roleConstructor := func(n string) forms.Field {
		return forms.NewTextField(n, forms.Trim)
	}
	roleConverter := func(values []forms.Field) interface{} {
		res := make(db.Strings, len(values))
		for i, x := range values {
			res[i] = x.String()
		}
		return res
	}

	// Only present policies that the current user can access
	choices := [][2]string{}
	for _, r := range availableScopes {
		if acls.InGroup(r[0], user.Group) {
			choices = append(choices, r)
		}
	}

	f := forms.NewListField("roles", roleConstructor, roleConverter)
	f.(*forms.ListField).SetChoices(choices)
	return f
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
				t.Roles = field.Value().(db.Strings)
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
