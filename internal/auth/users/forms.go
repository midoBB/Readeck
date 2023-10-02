// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package users

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"

	"codeberg.org/readeck/readeck/internal/acls"
	"codeberg.org/readeck/readeck/internal/db/types"
	"codeberg.org/readeck/readeck/pkg/forms"
)

type (
	ctxUserFormKey struct{}
)

var (
	// rxUsername is the regexp used to validate a username.
	rxUsername = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

var availableScopes = [][2]string{
	{"scoped_bookmarks_r", "Bookmarks : Read Only"},
	{"scoped_bookmarks_w", "Bookmarks : Write Only"},
	{"scoped_admin_r", "Admin : Read Only"},
	{"scoped_admin_w", "Admin : Write Only"},
}

// IsValidPassword is the password validation rule.
var IsValidPassword = forms.StringValidator(func(v string) bool {
	if strings.TrimSpace(v) == "" {
		return false
	}
	return len(v) >= 8
}, errors.New("password must be at least 8 character long"))

// IsValidUsername is the username validation rule.
var IsValidUsername = forms.StringValidator(func(v string) bool {
	return rxUsername.MatchString(v)
}, errors.New(`must contain English letters, digits, "_" and "-" only`))

// UserForm is the form used for user creation and update.
type UserForm struct {
	*forms.Form
}

// NewUserForm returns a UserForm instance.
func NewUserForm() *UserForm {
	return &UserForm{forms.Must(
		forms.NewTextField("username", forms.Trim, forms.Required),
		forms.NewTextField("password", forms.Required),
		forms.NewTextField("email", forms.Trim, forms.Required, forms.IsEmail),
		forms.NewChoiceField("group", availableGroups, forms.Trim, forms.Required),
	)}
}

// SetUser adds a user to the form's context.
func (f *UserForm) SetUser(u *User) {
	ctx := context.WithValue(f.Context(), ctxUserFormKey{}, u)
	f.SetContext(ctx)

	f.Get("username").Set(u.Username)
	f.Get("email").Set(u.Email)
	f.Get("group").Set(u.Group)
}

// Bind prepares the form before data binding.
// It changes some validators in case of user update.
func (f *UserForm) Bind() {
	f.Form.Bind()

	u, _ := f.Context().Value(ctxUserFormKey{}).(*User)
	if u == nil {
		// set default group
		f.Get("group").Set("user")
		return
	}

	// if we have a user, let some fields be optional
	f.Get("username").SetValidators(forms.Trim, forms.RequiredOrNil)
	f.Get("password").SetValidators()
	f.Get("email").SetValidators(forms.Trim, forms.RequiredOrNil, forms.IsEmail)
	f.Get("group").SetValidators(forms.Trim, forms.RequiredOrNil)
}

// Validate performs extra form validation.
func (f *UserForm) Validate() {
	f.AddErrors("password", forms.ValidateField(f.Get("password"), func(field forms.Field) error {
		if field.IsBound() && field.String() != "" && strings.TrimSpace(field.String()) == "" {
			return errors.New("cannot be empty")
		}
		return nil
	})...)

	u, _ := f.Context().Value(ctxUserFormKey{}).(*User)

	userQuery := Users.Query().
		Where(goqu.C("username").Eq(f.Get("username").String()))
	emailQuery := Users.Query().
		Where(goqu.C("email").Eq(f.Get("email").String()))

	if u != nil {
		userQuery = userQuery.Where(goqu.C("id").Neq(u.ID))
		emailQuery = emailQuery.Where(goqu.C("id").Neq(u.ID))
	}

	// Check that username is not already in use
	if c, err := userQuery.Count(); err != nil {
		f.AddErrors("", errors.New("validation process error"))
	} else if c > 0 {
		f.AddErrors("username", errors.New("username is already in use"))
	}

	// Check that email is not already in use
	if c, err := emailQuery.Count(); err != nil {
		f.AddErrors("", errors.New("validation process error"))
	} else if c > 0 {
		f.AddErrors("email", errors.New("email address is already in use"))
	}
}

// CreateUser performs the user creation.
func (f *UserForm) CreateUser() (*User, error) {
	u := &User{
		Username: f.Get("username").String(),
		Email:    f.Get("email").String(),
		Password: f.Get("password").String(),
		Group:    f.Get("group").String(),
	}

	err := Users.Create(u)
	if err != nil {
		f.AddErrors("", forms.ErrUnexpected)
	}

	return u, err
}

// UpdateUser performs a user update and returns a mapping of
// updated fields.
func (f *UserForm) UpdateUser(u *User) (res map[string]interface{}, err error) {
	if !f.IsBound() {
		err = errors.New("form is not bound")
		return
	}

	res = make(map[string]interface{})
	for _, field := range f.Fields() {
		switch field.Name() {
		case "password":
			if field.IsNil() || strings.TrimSpace(field.String()) == "" {
				continue
			}
			p, err := u.HashPassword(field.String())
			if err != nil {
				f.AddErrors("", forms.ErrUnexpected)
				return nil, err
			}
			res[field.Name()] = p
		default:
			if field.IsBound() && !field.IsNil() {
				res[field.Name()] = field.Value()
			}
		}
	}

	if len(res) > 0 {
		res["updated"] = time.Now()
		res["seed"] = u.SetSeed()
		if err = u.Update(res); err != nil {
			f.AddErrors("", forms.ErrUnexpected)
			return
		}
		if _, ok := res["password"]; ok {
			res["password"] = "-"
		}
	}
	res["id"] = u.ID
	delete(res, "seed")
	return
}

func NewRolesField(user *User) forms.Field {
	roleConstructor := func(n string) forms.Field {
		return forms.NewTextField(n, forms.Trim)
	}
	roleConverter := func(values []forms.Field) interface{} {
		res := make(types.Strings, len(values))
		for i, x := range values {
			res[i] = x.String()
		}
		return res
	}

	// Only present policies that the current user can access
	choices := [][2]string{}
	for _, r := range availableScopes {
		if user == nil || acls.InGroup(r[0], user.Group) {
			choices = append(choices, r)
		}
	}

	f := forms.NewListField("roles", roleConstructor, roleConverter)
	f.(*forms.ListField).SetChoices(choices)
	return f
}
