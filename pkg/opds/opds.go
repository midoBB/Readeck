// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package opds

import (
	"encoding/xml"
	"io"
	"time"

	"codeberg.org/readeck/readeck/pkg/bleach"
	"github.com/google/uuid"
)

const (
	// OPDSTypeNavigation is the link type for navigation
	OPDSTypeNavigation = "application/atom+xml; profile=opds-catalog; kind=navigation"
	// OPDSTypeAcquisistion is the link type for acquisition
	OPDSTypeAcquisistion = "application/atom+xml; profile=opds-catalog; kind=acquisition"
)

// Feed root element for acquisition or navigation feed
type Feed struct {
	XMLName      xml.Name `xml:"feed"`
	XMLns        string   `xml:"xmlns,attr"`
	XMLnsDC      string   `xml:"xmlns:dc,attr"`
	Lang         string   `xml:"xml:lang,attr"`
	ID           UUID     `xml:"id"`
	Title        string   `xml:"title"`
	Author       []Author `xml:"author,omitempty"`
	Updated      Time     `xml:"updated"`
	Entries      []Entry  `xml:"entry"`
	Links        []Link   `xml:"link"`
	TotalResults int      `xml:"totalResults,omitempty"`
	ItemsPerPage int      `xml:"itemsPerPage,omitempty"`
	FeedType     string   `xml:"-"`
}

// Link link to different resources
type Link struct {
	Rel                 string                `xml:"rel,attr"`
	Href                string                `xml:"href,attr"`
	TypeLink            string                `xml:"type,attr,omitempty"`
	Title               string                `xml:"title,attr,omitempty"`
	FacetGroup          string                `xml:"facetGroup,attr,omitempty"`
	Count               int                   `xml:"count,attr,omitempty"`
	IndirectAcquisition []IndirectAcquisition `xml:"indirectAcquisition,omitempty"`
}

// Author represent the feed author or the entry author
type Author struct {
	Name string `xml:"name"`
	URI  string `xml:"uri"`
}

// Entry an atom entry in the feed
type Entry struct {
	Title      string     `xml:"title"`
	ID         UUID       `xml:"id"`
	Identifier UUID       `xml:"dc:identifier"`
	Updated    Time       `xml:"updated"`
	Rights     string     `xml:"rights,omitempty"`
	Publisher  string     `xml:"dc:publisher,omitempty"`
	Author     []Author   `xml:"author,omitempty"`
	Language   string     `xml:"dc:language,omitempty"`
	Issued     *Time      `xml:"dc:issued,omitempty"`
	Published  *Time      `xml:"published,omitempty"`
	Category   []Category `xml:"category,omitempty"`
	Links      []Link     `xml:"link,omitempty"`
	Summary    *Content   `xml:"summary,omitempty"`
	Content    *Content   `xml:"content,omitempty"`
	Series     []Serie    `xml:"Series"`
}

// Content content tag in an entry, the type will be html or text
type Content struct {
	Content     string `xml:",cdata"`
	ContentType string `xml:"type,attr"`
}

// Category represent the book category with scheme and term to machine
// handling
type Category struct {
	Scheme string `xml:"scheme,attr"`
	Term   string `xml:"term,attr"`
	Label  string `xml:"label,attr"`
}

// IndirectAcquisition represent the link mostly for buying or borrowing
// a book
type IndirectAcquisition struct {
	TypeAcquisition     string                `xml:"type,attr"`
	IndirectAcquisition []IndirectAcquisition `xml:"indirectAcquisition"`
}

// Serie store serie information from schema.org
type Serie struct {
	Name     string  `xml:"name,attr"`
	URL      string  `xml:"url,attr"`
	Position float32 `xml:"position,attr"`
}

// NewLinkEntry creates a navigation link
func NewLinkEntry(title string, updated time.Time, href string) Entry {
	return Entry{
		Title:      bleach.SanitizeString(title),
		Updated:    *AtomDate(updated),
		Content:    &Content{ContentType: "text", Content: bleach.SanitizeString(title)},
		ID:         URLID(href),
		Identifier: URLID(href),
		Links: []Link{
			{
				Rel:      "subsection",
				Href:     href,
				TypeLink: OPDSTypeAcquisistion,
			},
		},
	}
}

// UUID is a wrapper around uuid.UUID that can be marshaled into
// a "urn:uuid" identifier.
type UUID struct {
	uuid.UUID
}

// ID returns a new UUID instance
func ID(src uuid.UUID) UUID {
	return UUID{src}
}

// URLID returns a new UUID instance, based on a URL.
func URLID(src string) UUID {
	return ID(uuid.NewMD5(uuid.NameSpaceURL, []byte(src)))
}

// MarshalText encoded an UUID into a "urn:uuid" format.
func (id UUID) MarshalText() ([]byte, error) {
	return []byte("urn:uuid:" + id.String()), nil
}

// Time is wrapper around time.Time that can marshal into an RFC3339 format.
type Time struct {
	time.Time
}

// AtomDate returns a new Time instance.
func AtomDate(src time.Time) *Time {
	return &Time{src}
}

// MarshalText encodes the Time instance in RFC3339 format.
func (d Time) MarshalText() ([]byte, error) {
	return []byte(d.Format(time.RFC3339)), nil
}

// Encode encodes a feed on a writer.
func (f *Feed) Encode(w io.Writer) (err error) {
	if _, err = io.WriteString(w, xml.Header); err != nil {
		return
	}

	f.Lang = "en"
	f.XMLns = "http://www.w3.org/2005/Atom"
	f.XMLnsDC = "http://purl.org/dc/terms/"

	enc := xml.NewEncoder(w)
	return enc.Encode(f)
}
