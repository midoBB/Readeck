package contentscripts_test

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"testing"

	"codeberg.org/readeck/readeck/pkg/extract/contentscripts"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestRequests(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "/json", func(r *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, map[string]any{
			"a": "testA",
			"b": "testB",
		})
	})
	httpmock.RegisterResponder("GET", "/text", func(r *http.Request) (*http.Response, error) {
		rsp := httpmock.NewBytesResponse(200, []byte("some text"))
		rsp.Header.Add("X-Test", "abc")
		return rsp, nil
	})
	httpmock.RegisterResponder("GET", "/error", func(r *http.Request) (*http.Response, error) {
		return httpmock.NewBytesResponse(400, []byte("error")), nil
	})
	httpmock.RegisterResponder("GET", "/echo", func(r *http.Request) (*http.Response, error) {
		return httpmock.NewJsonResponse(200, map[string]any{
			"url":     r.URL.String(),
			"headers": r.Header,
		})
	})
	httpmock.RegisterResponder("POST", "/echo", func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)

		return httpmock.NewJsonResponse(200, map[string]any{
			"url":     r.URL.String(),
			"headers": r.Header,
			"body":    string(body),
		})
	})

	tests := []struct {
		src      string
		expected any
		error    error
	}{
		{
			`
			let rsp = requests.get("http://example.net/json")
			rsp.raiseForStatus()
			rsp.text()
			`,
			`{"a":"testA","b":"testB"}`,
			nil,
		},
		{
			`
			let rsp = requests.get("http://example.net/json")
			rsp.raiseForStatus()
			rsp.json()
			`,
			map[string]any{"a": "testA", "b": "testB"},
			nil,
		},
		{
			`
			let rsp = requests.get("http://example.net/text")
			rsp.raiseForStatus()
			rsp.status
			`,
			int64(200),
			nil,
		},
		{
			`
			let rsp = requests.get("http://example.net/text")
			rsp.raiseForStatus()
			rsp.headers
			`,
			map[string]string{"X-Test": "abc"},
			nil,
		},
		{
			`
			let rsp = requests.get("http://example.net/text")
			rsp.raiseForStatus()
			rsp.text()
			`,
			"some text",
			nil,
		},
		{
			`
			let rsp = requests.get("http://example.net/text")
			rsp.raiseForStatus()
			rsp.json()
			`,
			"",
			errors.New("invalid character"),
		},
		{
			`
			let rsp = requests.get("http://example.net/error")
			rsp.raiseForStatus()
			rsp.json()
			`,
			"",
			errors.New("invalid status code 400"),
		},
		{
			`
			let rsp = requests.get("http://example.net/echo")
			rsp.raiseForStatus()
			rsp.json()
			`,
			map[string]any{
				"url":     "http://example.net/echo",
				"headers": map[string]any{},
			},
			nil,
		},
		{
			`
			let rsp = requests.get("http://example.net/echo", {
				"content-type": "application/json",
			})
			rsp.raiseForStatus()
			rsp.json()
			`,
			map[string]any{
				"url": "http://example.net/echo",
				"headers": map[string]any{
					"Content-Type": []any{"application/json"},
				},
			},
			nil,
		},
		{
			`
			let rsp = requests.post("http://example.net/echo", JSON.stringify({
				"test": "abc",
				"data": "xyz",
			}))
			rsp.raiseForStatus()
			rsp.json()
			`,
			map[string]any{
				"url":     "http://example.net/echo",
				"headers": map[string]any{},
				"body":    `{"test":"abc","data":"xyz"}`,
			},
			nil,
		},
		{
			`
			let rsp = requests.post("http://example.net/echo", JSON.stringify({
				"test": "abc",
				"data": "xyz",
			}), {
				"content-type": "application/json",
			})
			rsp.raiseForStatus()
			rsp.json()
			`,
			map[string]any{
				"url": "http://example.net/echo",
				"headers": map[string]any{
					"Content-Type": []any{"application/json"},
				},
				"body": `{"test":"abc","data":"xyz"}`,
			},
			nil,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			vm := contentscripts.New()
			vm.Set("requests", contentscripts.NewHTTPClient(vm, http.DefaultClient))
			v, err := vm.RunString(test.src)
			if test.error == nil {
				if assert.NoError(t, err) {
					assert.Equal(t, test.expected, v.Export())
				}
			} else {
				assert.Error(t, err)
				assert.ErrorContains(t, err, test.error.Error())
			}
		})
	}
}
