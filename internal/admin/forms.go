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

func newDeleteForm() *deleteForm {
	return &deleteForm{forms.Must(
		forms.NewBooleanField("cancel"),
		forms.NewTextField("_to"),
	)}
}

// trigger launch the user deletion or cancel task.
func (f *deleteForm) trigger(u *users.User) {
	if !f.Get("cancel").IsNil() && f.Get("cancel").Value().(bool) {
		deleteUserTask.Cancel(u.ID)
		return
	}

	deleteUserTask.Run(u.ID, u.ID)
}
