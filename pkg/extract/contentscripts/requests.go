// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package contentscripts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dop251/goja"
)

type httpClient struct {
	*http.Client
	vm *Runtime
}

type httpResponse struct {
	*http.Response
	vm   *Runtime
	body *bytes.Buffer
}

// NewHTTPClient returns a new (very) simple HTTP client for the JS runtime.
func NewHTTPClient(vm *Runtime, client *http.Client) *goja.Object {
	c := &httpClient{vm: vm, Client: client}

	obj := vm.NewObject()
	obj.Set("get", c.get)
	obj.Set("post", c.post)

	return obj
}

func (c *httpClient) Do(req *http.Request, args ...goja.Value) (*goja.Object, error) {
	c.vm.GetLogger().
		WithField("method", req.Method).
		WithField("url", req.URL.String()).
		Debug("request")

	if len(args) > 0 {
		var headers map[string]string
		if err := c.vm.ExportTo(args[0], &headers); err != nil {
			return nil, err
		}
		for k, v := range headers {
			k = http.CanonicalHeaderKey(k)
			if k == "Content-Length" {
				continue
			}
			req.Header.Set(k, v)
		}
	}

	r, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	return newHTTPResponse(c.vm, r)
}

func (c *httpClient) get(url string, args ...goja.Value) (*goja.Object, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return c.Do(req, args...)
}

func (c *httpClient) post(url string, data []byte, args ...goja.Value) (*goja.Object, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	return c.Do(req, args...)
}

func newHTTPResponse(vm *Runtime, rsp *http.Response) (*goja.Object, error) {
	defer rsp.Body.Close()

	r := &httpResponse{vm: vm, Response: rsp, body: new(bytes.Buffer)}
	if _, err := io.Copy(r.body, rsp.Body); err != nil {
		return nil, err
	}

	headers := map[string]string{}
	for k, v := range r.Header {
		headers[k] = strings.Join(v, " ")
	}

	obj := vm.NewObject()
	obj.Set("status", r.StatusCode)
	obj.Set("headers", headers)
	obj.Set("raiseForStatus", r.raiseForStatus)
	obj.Set("json", r.json)
	obj.Set("text", r.text)

	return obj, nil
}

func (r *httpResponse) raiseForStatus() error {
	if r.StatusCode/100 != 2 {
		return fmt.Errorf("invalid status code %d", r.StatusCode)
	}
	return nil
}

func (r *httpResponse) json() (goja.Value, error) {
	var res any
	if err := json.Unmarshal(r.body.Bytes(), &res); err != nil {
		return nil, err
	}
	return r.vm.ToValue(res), nil
}

func (r *httpResponse) text() string {
	return r.body.String()
}
