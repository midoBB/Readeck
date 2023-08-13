// SPDX-FileCopyrightText: © 2020 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package extract

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"

	"github.com/go-shiori/dom"
	"github.com/readeck/readeck/pkg/glob"
	log "github.com/sirupsen/logrus"
)

type (
	// ProcessStep defines a type of process applied during extraction
	ProcessStep int

	// Processor is the process function
	Processor func(*ProcessMessage, Processor) Processor

	// ProcessList holds the processes that will be applied
	ProcessList []Processor

	// ProcessMessage holds the process message that is passed (and changed)
	// by the subsequent processes.
	ProcessMessage struct {
		Context   context.Context
		Extractor *Extractor
		Log       *log.Entry
		Dom       *html.Node

		position     int
		resetCounter int
		maxReset     int
		maxDrops     int
		step         ProcessStep
		canceled     bool
		values       map[string]interface{}
	}

	// ProxyMatcher describes a mapping of host/url for proxy dispatch.
	ProxyMatcher interface {
		// Returns the matching host
		Host() string
		// Returns the proxy URL
		URL() *url.URL
	}
)

const (
	// StepStart happens before the connection is made.
	StepStart ProcessStep = iota + 1

	// StepBody happens after receiving the resource body.
	StepBody

	// StepDom happens after parsing the resource DOM tree.
	StepDom

	// StepFinish happens at the very end of the extraction.
	StepFinish

	// StepPostProcess happens after looping over each Drop.
	StepPostProcess
)

// Step returns the current process step
func (m *ProcessMessage) Step() ProcessStep {
	return m.step
}

// Position returns the current process position
func (m *ProcessMessage) Position() int {
	return m.position
}

// ResetPosition lets the process start over (normally with a new URL).
// It holds a counter and cancels everything after too many resets (defined by maxReset).
func (m *ProcessMessage) ResetPosition() {
	if m.resetCounter >= m.maxReset {
		m.Cancel("too many redirects")
	}
	m.resetCounter++
	m.position = -1
}

// Value returns a stored message value.
func (m *ProcessMessage) Value(name string) interface{} {
	return m.values[name]
}

// SetValue sets a new message value.
func (m *ProcessMessage) SetValue(name string, value interface{}) {
	m.values[name] = value
}

// ResetContent empty the message Dom and all the drops body
func (m *ProcessMessage) ResetContent() {
	m.Dom = nil
	m.Extractor.Drops()[m.position].Body = []byte{}
}

// Cancel fully cancel the extract process.
func (m *ProcessMessage) Cancel(reason string, args ...interface{}) {
	m.Log.WithError(fmt.Errorf(reason, args...)).Error("operation canceled")
	m.canceled = true
}

// Error holds all the non-fatal errors that were
// caught during extraction.
type Error []error

func (e Error) Error() string {
	s := make([]string, len(e))
	for i, err := range e {
		s[i] = err.Error()
	}
	return strings.Join(s, ", ")
}

// URLList hold a list of URLs
type URLList map[string]bool

// Add adds a new URL to the list
func (l URLList) Add(v *url.URL) {
	c := *v
	c.Fragment = ""
	l[c.String()] = true
}

// IsPresent returns
func (l URLList) IsPresent(v *url.URL) bool {
	c := *v
	c.Fragment = ""
	return l[c.String()]
}

// Extractor is a page extractor.
type Extractor struct {
	URL       *url.URL
	HTML      []byte
	Text      string
	Visited   URLList
	Logs      []string
	Context   context.Context
	LogFields *log.Fields

	client          *http.Client
	processors      ProcessList
	errors          Error
	drops           []*Drop
	cachedResources map[string]*cachedResource
}

// New returns an Extractor instance for a given URL,
// with a default HTTP client.
func New(src string, options ...func(e *Extractor)) (*Extractor, error) {
	URL, err := url.Parse(src)
	if err != nil {
		return nil, err
	}
	URL.Fragment = ""

	res := &Extractor{
		URL:             URL,
		Visited:         URLList{},
		Context:         context.TODO(),
		client:          NewClient(),
		cachedResources: make(map[string]*cachedResource),
		processors:      ProcessList{},
		drops:           []*Drop{NewDrop(URL)},
	}

	t := res.client.Transport.(*Transport)
	t.SetRoundTripper(res.getFromCache)

	for _, fn := range options {
		if fn != nil {
			fn(res)
		}
	}

	return res, nil
}

// SetLogFields sets the default log fields for the extractor.
func SetLogFields(f *log.Fields) func(e *Extractor) {
	return func(e *Extractor) {
		e.LogFields = f
	}
}

// SetDeniedIPs sets a list of ip or cird that cannot be reached
// by the extraction client.
func SetDeniedIPs(netList []*net.IPNet) func(e *Extractor) {
	return func(e *Extractor) {
		if t, ok := e.client.Transport.(*Transport); ok {
			t.deniedIPs = netList
		}
	}
}

// SetProxyList adds a new proxy dispatcher function to the HTTP transport.
func SetProxyList(list []ProxyMatcher) func(e *Extractor) {
	return func(e *Extractor) {
		t := e.client.Transport.(*Transport)
		htr := t.tr.(*http.Transport)
		htr.Proxy = func(r *http.Request) (*url.URL, error) {
			for _, p := range list {
				if glob.Glob(p.Host(), r.URL.Host) {
					e.GetLogger().WithField("proxy", p.URL().String()).Debug("using proxy")
					return p.URL(), nil
				}
			}
			return nil, nil
		}
	}
}

// Client returns the extractor's HTTP client.
func (e *Extractor) Client() *http.Client {
	return e.client
}

// AddToCache adds a resource to the extractor's resource cache.
// The cache will be used by the HTTP client during its round trip.
func (e *Extractor) AddToCache(url string, headers map[string]string, body io.ReadSeeker) {
	e.cachedResources[url] = &cachedResource{headers: headers, data: &cacheEntry{body}}
}

// IsInCache returns true if a given URL is present in the
// resource cache mapping.
func (e *Extractor) IsInCache(url string) bool {
	_, ok := e.cachedResources[url]
	return ok
}

// Errors returns the extractor's error list.
func (e *Extractor) Errors() Error {
	return e.errors
}

// AddError add a new error to the extractor's error list.
func (e *Extractor) AddError(err error) {
	e.errors = append(e.errors, err)
}

// Drops returns the extractor's drop list.
func (e *Extractor) Drops() []*Drop {
	return e.drops
}

// Drop return the extractor's first drop, when there is one.
func (e *Extractor) Drop() *Drop {
	if len(e.drops) == 0 {
		return nil
	}
	return e.drops[0]
}

// AddDrop adds a new Drop to the drop list.
func (e *Extractor) AddDrop(src *url.URL) {
	e.drops = append(e.drops, NewDrop(src))
}

// ReplaceDrop replaces the main Drop with a new one.
func (e *Extractor) ReplaceDrop(src *url.URL) error {
	if len(e.drops) != 1 {
		return errors.New("cannot replace a drop when there are more that one")
	}

	e.drops[0] = NewDrop(src)
	return nil
}

// AddProcessors adds extract processor(s) to the list
func (e *Extractor) AddProcessors(p ...Processor) {
	e.processors = append(e.processors, p...)
}

// NewProcessMessage returns a new ProcessMessage for a given step.
func (e *Extractor) NewProcessMessage(step ProcessStep) *ProcessMessage {
	logEntry := log.NewEntry(e.GetLogger())
	if e.LogFields != nil {
		logEntry = logEntry.WithFields(*e.LogFields)
	}

	return &ProcessMessage{
		Extractor:    e,
		Log:          logEntry,
		step:         step,
		resetCounter: 0,
		maxReset:     10,
		maxDrops:     100,
		values:       make(map[string]interface{}),
	}
}

// GetLogger returns a logger for the extractor.
// This standard logger will copy everything to the
// extractor Log slice.
func (e *Extractor) GetLogger() *log.Logger {
	logger := log.New()
	logger.Formatter = log.StandardLogger().Formatter
	logger.Level = log.DebugLevel
	logger.SetOutput(ioutil.Discard)
	logger.AddHook(&messageLogHook{e})

	return logger
}

// Run start the extraction process.
func (e *Extractor) Run() {
	i := 0
	m := e.NewProcessMessage(0)

	for i < len(e.drops) {
		d := e.drops[i]

		// Don't visit the same URL twice
		if e.Visited.IsPresent(d.URL) {
			i++
			continue
		}
		e.Visited.Add(d.URL)

		// Don't let any page fool us into processing an
		// unlimited number of pages.
		if len(e.drops) >= m.maxDrops {
			m.Cancel("too many pages")
		}

		m.position = i

		// Start extraction
		m.Log.WithField("idx", i).WithField("url", d.URL.String()).Info("start")
		m.step = StepStart
		e.runProcessors(m)
		if m.canceled {
			return
		}

		err := d.Load(e.client)
		if err != nil {
			m.Log.WithError(err).Error("cannot load resource")
			return
		}

		// First process pass
		m.Log.Debug("step body")
		m.step = StepBody
		e.runProcessors(m)
		if m.canceled {
			return
		}

		// Load the dom
		if d.IsHTML() {
			func() {
				doc, err := html.Parse(bytes.NewReader(d.Body))
				defer func() {
					m.Dom = nil
				}()

				if err != nil {
					m.Log.WithError(err).Error("cannot parse resource")
					return
				}

				m.Log.Debug("step DOM")
				m.Dom = doc
				m.step = StepDom
				e.runProcessors(m)
				if m.canceled {
					return
				}

				// Render the final document body
				if m.Dom != nil {
					buf := bytes.NewBuffer(nil)
					html.Render(buf, convertBodyNodes(m.Dom))
					d.Body = buf.Bytes()
				}
			}()
		}

		// Final processes
		m.Log.Debug("step finish")
		m.step = StepFinish
		e.runProcessors(m)
		if m.canceled {
			return
		}

		// A processor can change the position in the loop
		i = m.position + 1
	}

	// Postprocess
	m.Log.Debug("postprocess")
	m.step = StepPostProcess
	e.setFinalHTML()
	e.runProcessors(m)
}

func (e *Extractor) getFromCache(req *http.Request) (*http.Response, error) {
	u := req.URL.String()
	entry, ok := e.cachedResources[u]
	if !ok {
		return nil, nil
	}

	e.GetLogger().WithField("url", u).Debug("cache hit")
	headers := make(http.Header)
	for k, v := range entry.headers {
		headers.Set(k, v)
	}

	return &http.Response{
		Status:        "OK",
		StatusCode:    http.StatusOK,
		Header:        headers,
		Body:          entry.data,
		Request:       req,
		ContentLength: -1,
	}, nil
}

func (e *Extractor) runProcessors(m *ProcessMessage) {
	if e.processors == nil || len(e.processors) == 0 {
		return
	}

	p := e.processors[0]
	i := 0
	for {
		var next Processor
		i++
		if i < len(e.processors) {
			next = e.processors[i]
		}
		p = p(m, next)
		if p == nil {
			return
		}
	}
}

// convertBodyNodes extracts all the element from a
// document body and then returns a new HTML Document
// containing only the body's children.
func convertBodyNodes(top *html.Node) *html.Node {
	doc := &html.Node{
		Type: html.DocumentNode,
	}
	for _, node := range dom.GetElementsByTagName(top, "body") {
		for _, c := range dom.ChildNodes(node) {
			doc.AppendChild(dom.Clone(c, true))
		}
	}

	return doc
}

func (e *Extractor) setFinalHTML() {
	buf := &bytes.Buffer{}
	for i, d := range e.drops {
		if len(d.Body) == 0 {
			continue
		}
		fmt.Fprintf(buf, "<!-- page %d -->\n", i+1)
		buf.Write(d.Body)
		buf.WriteString("\n")
	}
	e.HTML = buf.Bytes()
}

type cachedResource struct {
	headers map[string]string
	data    io.ReadCloser
}

type cacheEntry struct {
	body io.ReadSeeker
}

func (cr *cacheEntry) Read(p []byte) (n int, err error) {
	n, err = cr.body.Read(p)
	if err == io.EOF {
		cr.body.Seek(0, 0)
	}
	return n, err
}

func (cr *cacheEntry) Close() error {
	cr.body.Seek(0, 0)
	return nil
}
