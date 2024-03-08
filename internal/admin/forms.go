// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package admin

import (
	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/pkg/forms"
)

type deleteForm struct {
	*forms.Form
}

func newDeleteForm(tr forms.Translator) (f *deleteForm) {
	f = &deleteForm{forms.Must(
		forms.NewBooleanField("cancel"),
		forms.NewTextField("_to"),
	)}
	f.SetLocale(tr)
	return
}

// trigger launch the user deletion or cancel task.
func (f *deleteForm) trigger(u *users.User) error {
	if !f.Get("cancel").IsNil() && f.Get("cancel").Value().(bool) {
		return deleteUserTask.Cancel(u.ID)
	}

	return deleteUserTask.Run(u.ID, u.ID)
}
