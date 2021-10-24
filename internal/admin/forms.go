package admin

import (
	"github.com/readeck/readeck/internal/auth/users"
	"github.com/readeck/readeck/pkg/forms"
)

type deleteForm struct {
	*forms.Form
}

func newDeleteForm() *deleteForm {
	return &deleteForm{forms.Must(
		forms.NewBooleanField("cancel"),
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
