package admin_test

import (
	"fmt"
	"testing"

	. "github.com/readeck/readeck/internal/testing"
)

func TestPermissions(t *testing.T) {
	app := NewTestApp(t)
	defer func() {
		app.Close(t)
	}()

	client := NewClient(t, app)
	u1, err := NewTestUser("test1", "test1@localhost", "test1", "user")
	if err != nil {
		t.Fatal(err)
	}
	u2, err := NewTestUser("test2", "test2@localhost", "test2", "user")
	if err != nil {
		t.Fatal(err)
	}

	users := []string{"admin", "staff", "user", "disabled", ""}
	for _, user := range users {
		RunRequestSequence(t, client, user,
			// API
			RequestTest{
				JSON:   true,
				Target: "/api/admin/users",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin":
						r.AssertStatus(t, 200)
					case "":
						r.AssertStatus(t, 401)
					default:
						r.AssertStatus(t, 403)
					}
				},
			},
			RequestTest{
				Method: "POST",
				Target: "/api/admin/users",
				JSON:   map[string]string{},
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin":
						r.AssertStatus(t, 422)
					case "":
						r.AssertStatus(t, 401)
					default:
						r.AssertStatus(t, 403)
					}
				},
			},
			RequestTest{
				JSON:   true,
				Target: fmt.Sprintf("/api/admin/users/%d", u1.User.ID),
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin":
						r.AssertStatus(t, 200)
					case "":
						r.AssertStatus(t, 401)
					default:
						r.AssertStatus(t, 403)
					}
				},
			},
			RequestTest{
				Method: "PATCH",
				Target: fmt.Sprintf("/api/admin/users/%d", u1.User.ID),
				JSON:   map[string]string{},
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin":
						r.AssertStatus(t, 200)
					case "":
						r.AssertStatus(t, 401)
					default:
						r.AssertStatus(t, 403)
					}
				},
			},
			RequestTest{
				Method: "DELETE",
				Target: fmt.Sprintf("/api/admin/users/%d", u1.User.ID),
				JSON:   true,
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin":
						r.AssertStatus(t, 204)
					case "":
						r.AssertStatus(t, 401)
					default:
						r.AssertStatus(t, 403)
					}
				},
			},

			// Views
			RequestTest{
				Target: "/admin",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/admin/users")
					case "":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/login")
					default:
						r.AssertStatus(t, 403)
					}
				},
			},
			RequestTest{
				Target: "/admin/users",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin":
						r.AssertStatus(t, 200)
					case "":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/login")
					default:
						r.AssertStatus(t, 403)
					}
				},
			},
			RequestTest{
				Target: "/admin/users/add",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin":
						r.AssertStatus(t, 200)
					case "":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/login")
					default:
						r.AssertStatus(t, 403)
					}
				},
			},
			RequestTest{
				Method: "POST",
				Target: "/admin/users/add",
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin":
						r.AssertStatus(t, 200)
					case "":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/login")
					default:
						r.AssertStatus(t, 403)
					}
				},
			},
			RequestTest{
				Target: fmt.Sprintf("/admin/users/%d", u2.User.ID),
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin":
						r.AssertStatus(t, 200)
					case "":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/login")
					default:
						r.AssertStatus(t, 403)
					}
				},
			},
			RequestTest{
				Method: "POST",
				Target: fmt.Sprintf("/admin/users/%d/delete", u2.User.ID),
				Assert: func(t *testing.T, r *Response) {
					switch user {
					case "admin":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, fmt.Sprintf("/admin/users/%d", u2.User.ID))
					case "":
						r.AssertStatus(t, 303)
						r.AssertRedirect(t, "/login")
					default:
						r.AssertStatus(t, 403)
					}
				},
			},
		)
	}
}
