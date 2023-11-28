// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package testing provides tools to tests the HTTP routes, the message bus, email sending, etc.
package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"text/template"

	"golang.org/x/net/html"

	"github.com/go-shiori/dom"
	"github.com/kinbiko/jsonassert"
	"github.com/stretchr/testify/assert"
	mail "github.com/xhit/go-simple-mail/v2"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/internal/app"
	"codeberg.org/readeck/readeck/internal/auth/tokens"
	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/internal/db"
	"codeberg.org/readeck/readeck/internal/email"
	"codeberg.org/readeck/readeck/internal/server"
)

// TestUser contains the user data that we can use during tests.
type TestUser struct {
	User     *users.User
	Token    *tokens.Token
	password string
	jwt      string
}

// NewTestUser creates a new user for testing.
func NewTestUser(name, email, password, group string) (*TestUser, error) {
	u := &users.User{
		Username: name,
		Email:    email,
		Password: password,
		Group:    group,
	}
	if err := users.Users.Create(u); err != nil {
		return nil, err
	}

	res := &TestUser{User: u, password: password}

	res.Token = &tokens.Token{
		UserID:      &u.ID,
		IsEnabled:   true,
		Application: "tests",
	}
	if err := tokens.Tokens.Create(res.Token); err != nil {
		return nil, err
	}
	jwt, err := tokens.NewJwtToken(res.Token.UID)
	if err != nil {
		return nil, err
	}
	res.jwt = jwt.String()

	return res, nil
}

// Password returns the user's password.
func (tu *TestUser) Password() string {
	return tu.password
}

// JWT returns the user's API token.
func (tu *TestUser) JWT() string {
	return tu.jwt
}

// Login performs the login for the user.
func (tu *TestUser) Login(c *Client) {
	c.Login(tu.User.Username, tu.Password())
}

// TestApp holds information of the application for testing.
type TestApp struct {
	TmpDir    string
	Srv       *server.Server
	Users     map[string]*TestUser
	LastEmail string
}

// NewTestApp initializes TestApp with a default configuration,
// some users, and an http muxer ready to accept requests.
func NewTestApp(t *testing.T) *TestApp {
	var err error
	tmpDir, err := os.MkdirTemp(os.TempDir(), "readeck_*")
	if err != nil {
		t.Fatal(err)
	}

	configs.Config.Main.SecretKey = "1234567890"
	configs.Config.Main.DataDirectory = tmpDir
	configs.Config.Main.DevMode = false
	configs.Config.Main.LogLevel = "error"
	configs.Config.Database.Source = "sqlite3::memory:"
	configs.Config.Server.AllowedHosts = []string{"readeck.example.org"}

	configs.InitConfiguration()

	app.InitApp()
	configs.Config.Commissioned = true

	// Init test app
	ta := &TestApp{
		TmpDir: tmpDir,
		Users:  make(map[string]*TestUser),
	}

	userList := map[string]string{
		"admin":    "admin",
		"user":     "user",
		"staff":    "staff",
		"disabled": "none",
	}

	_, err = db.Q().Delete(users.TableName).Executor().Exec()
	if err != nil {
		t.Fatal(err)
	}

	for name, group := range userList {
		tu, err := NewTestUser(name, name+"@localhost", name, group)
		if err != nil {
			t.Fatal(err)
		}
		ta.Users[name] = tu
	}

	// Email sender
	configs.Config.Email.Host = "localhost"
	email.Sender = ta

	// Start event manager
	startEventManager()

	// Init test server
	ta.Srv = server.New(configs.Config.Server.Prefix)
	err = app.InitServer(ta.Srv)
	if err != nil {
		t.Fatal(err)
	}

	return ta
}

// Close removes artifacts that were needed for testing.
func (ta *TestApp) Close(t *testing.T) {
	if err := db.Close(); err != nil {
		t.Logf("error closing database: %s", err)
	}
	if err := os.RemoveAll(ta.TmpDir); err != nil {
		t.Logf("error removing temporary folder: %s", err)
	}

	// Reset the bus
	Events().Stop()
	Store().Clear()
}

// SendEmail implements email.sender interface and stores the last sent message.
func (ta *TestApp) SendEmail(m *mail.Email) error {
	ta.LastEmail = m.GetMessage()
	return nil
}

// Client is a thin HTTP client over the main server router.
type Client struct {
	*testing.T
	app       *TestApp
	URL       *url.URL
	Jar       http.CookieJar
	CsrfToken string
}

// NewClient creates a new Client instance.
func NewClient(t *testing.T, app *TestApp) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		T:   t,
		app: app,
		URL: &url.URL{Scheme: "https", Host: "readeck.example.org"},
		Jar: jar,
	}
}

// NewRequest returns a new http.Request instance ready for tests.
func (c *Client) NewRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.URL.Host = c.URL.Host
	req.URL.Scheme = c.URL.Scheme
	req.Host = c.URL.Host

	// Set request cookies
	for _, cookie := range c.Jar.Cookies(req.URL) {
		req.AddCookie(cookie)
	}

	return req
}

// NewFormRequest returns a new http.Request instance to be used for sending form data.
func (c *Client) NewFormRequest(method, target string, data url.Values) *http.Request {
	if c.CsrfToken != "" {
		data.Set("__csrf__", c.CsrfToken)
	}

	req := c.NewRequest(method, target, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// NewJSONRequest returns a new http.Request instance to be used for sending and receiving
// JSON data.
func (c *Client) NewJSONRequest(method, target string, data interface{}) *http.Request {
	var body io.Reader
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			c.Fatal("unable to marshal data")
		}
		body = bytes.NewReader(b)
	}

	req := c.NewRequest(method, target, body)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

// Request performs a request using httptest tools.
// It returns a Response instance that can be evaluated for testing
// purposes.
func (c *Client) Request(req *http.Request) *Response {
	w := httptest.NewRecorder()

	// Perform request
	c.app.Srv.Router.ServeHTTP(w, req)

	// Update cookies from response
	//nolint:bodyclose
	if rc := w.Result().Cookies(); len(rc) > 0 {
		c.Jar.SetCookies(req.URL, rc)
	}

	// Prepare response instance
	rsp, err := NewResponse(w, req)
	if err != nil {
		c.Fatal(err)
	}
	c.CsrfToken = rsp.CsrfToken

	return rsp
}

// Login logs in the given user.
func (c *Client) Login(username, password string) {
	r := c.Get("/login")
	if r.StatusCode != http.StatusOK {
		c.Fatalf("Invalid status %d", r.StatusCode)
	}

	data := url.Values{}
	data.Add("username", username)
	data.Add("password", password)
	r = c.PostForm("/login", data)
	if r.StatusCode != http.StatusSeeOther {
		c.Fatalf("Invalid status %d", r.StatusCode)
	}
}

// Logout empties the client's cookie jar.
func (c *Client) Logout() {
	c.Jar, _ = cookiejar.New(nil)
}

// Cookies returns the stored cookie for the current client session.
func (c *Client) Cookies() []*http.Cookie {
	return c.Jar.Cookies(c.URL)
}

// Get performs a GET request on the given path.
func (c *Client) Get(target string) *Response {
	return c.Request(c.NewRequest("GET", target, nil))
}

// PostForm performs a POST request on the given path with some data.
// If available, the CSRF token is automaticaly sent with the data.
func (c *Client) PostForm(target string, data url.Values) *Response {
	return c.Request(c.NewFormRequest("POST", target, data))
}

// RequestJSON performs a JSON HTTP requests (sending and receiving data).
func (c *Client) RequestJSON(method string, target string, data interface{}) *Response {
	return c.Request(c.NewJSONRequest(method, target, data))
}

// RenderTemplate executes a template string using some properties of the client.
// (Users, URL). Extra data can be sent using the extra map.
func (c *Client) RenderTemplate(src string, extra map[string]interface{}) (string, error) {
	tpl, err := template.New("").Parse(src)
	if err != nil {
		return "", err
	}
	buf := bytes.Buffer{}
	data := map[string]interface{}{
		"Users": c.app.Users,
		"URL":   fmt.Sprintf("http://%s", c.URL.Host),
	}
	for k, v := range extra {
		data[k] = v
	}

	if err = tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Response is a wrapper around http.Response where the body is stored and
// the HTML (when applicable) is parsed in advance.
type Response struct {
	*http.Response
	URL       *url.URL
	Redirect  string
	Body      []byte
	HTML      *html.Node
	CsrfToken string
}

// NewResponse returns a Response instance based on the ResponseRecorder
// given in input.
func NewResponse(rec *httptest.ResponseRecorder, req *http.Request) (*Response, error) {
	var err error
	r := &Response{Response: rec.Result()} //nolint:bodyclose

	u2 := new(url.URL)
	*u2 = *req.URL
	u2.Scheme = "http"

	r.URL = u2

	// Set redirect if any
	if loc := r.Header.Get("location"); loc != "" {
		redir, err := r.URL.Parse(loc)
		if err != nil {
			return nil, err
		}
		if redir.Host == u2.Host {
			redir.Scheme = ""
			redir.Host = ""
		}
		r.Redirect = redir.String()
	}

	// Read the response's body
	r.Body, err = io.ReadAll(r.Response.Body)
	if err != nil {
		return nil, err
	}

	// When an HTML response is received, parse it
	if strings.HasPrefix(r.Response.Header.Get("content-type"), "text/html") {
		r.HTML, err = html.Parse(bytes.NewReader(r.Body))
		if err != nil {
			return nil, err
		}

		// Extract the CSRF token, we'll need it to post data
		n := dom.QuerySelector(r.HTML, `head>meta[name="x-csrf-token"]`)
		if n != nil {
			r.CsrfToken = dom.GetAttribute(n, "content")
		}

	}

	return r, nil
}

// Path returns the path and querystring of the response URL.
func (r *Response) Path() string {
	u := new(url.URL)
	*u = *r.URL
	u.Scheme = ""
	u.Host = ""
	return u.String()
}

// AssertStatus checks the response's expected status.
func (r *Response) AssertStatus(t *testing.T, expected int) {
	assert.Equal(t, expected, r.StatusCode)
}

// AssertRedirect checks that the expected target is present in a Location header.
func (r *Response) AssertRedirect(t *testing.T, expected string) {
	assert.Regexp(t, expected, r.Redirect)
}

// AssertJSON checks that the response's JSON matches what we expect.
func (r *Response) AssertJSON(t *testing.T, expected string) {
	jsonassert.New(t).Assertf(string(r.Body), expected)
	if t.Failed() {
		t.Errorf("Received JSON: %s\n", string(r.Body))
	}
}

// RequestTest contains data that are used to perform requests and assert some results.
type RequestTest struct {
	Method         string
	Target         string
	Form           url.Values
	JSON           interface{}
	ExpectStatus   int
	ExpectRedirect string
	ExpectJSON     string
	ExpectContains string
	Assert         func(*testing.T, *Response)
}

// RunRequestSequence performs a serie of requests using RequestTest instances.
func RunRequestSequence(t *testing.T, c *Client, user string, tests ...RequestTest) {
	if user != "" {
		c.app.Users[user].Login(c)
		defer c.Logout()
	} else {
		c.Logout()
	}

	// Empty the event queue after a sequence
	defer Events().Clear()

	t.Run(fmt.Sprintf("sequence (%s)", user), func(t *testing.T) {
		history := []*Response{}
		templateData := map[string]interface{}{
			"History": &history,
		}

		for _, test := range tests {
			target, err := c.RenderTemplate(test.Target, templateData)
			if err != nil {
				t.Error(err)
				return
			}
			if test.Method == "" {
				test.Method = "GET"
			}

			t.Run(fmt.Sprintf("%s %s", test.Method, target), func(t *testing.T) {
				var req *http.Request

				switch {
				case test.JSON != nil:
					var data interface{}
					switch test.Method {
					case "POST", "PATCH", "PUT":
						data = test.JSON
					default:
						data = nil
					}
					req = c.NewJSONRequest(test.Method, target, data)
					req.Header.Del("Cookie")
					if user != "" {
						req.Header.Set("Authorization", "Bearer "+c.app.Users[user].JWT())
					} else {
						req.Header.Del("Authorization")
					}

				case test.Form != nil || test.Method == "POST":
					if test.Form == nil {
						test.Form = url.Values{}
					}
					req = c.NewFormRequest(test.Method, target, test.Form)

				default:
					req = c.NewRequest(test.Method, target, nil)
				}

				// Perform request
				rsp := c.Request(req)

				// Add request to history before all the asserts
				history = append([]*Response{rsp}, history...)

				if test.ExpectStatus != 0 {
					rsp.AssertStatus(t, test.ExpectStatus)
				}

				if test.ExpectRedirect != "" {
					rsp.AssertRedirect(t, test.ExpectRedirect)
				}

				if test.ExpectJSON != "" {
					s, err := c.RenderTemplate(test.ExpectJSON, templateData)
					if err != nil {
						t.Error(err)
						return
					}
					rsp.AssertJSON(t, s)
				}

				if test.ExpectContains != "" {
					assert.Contains(t, string(rsp.Body), test.ExpectContains)
				}

				if test.Assert != nil {
					test.Assert(t, rsp)
				}
			})
		}
	})
}
