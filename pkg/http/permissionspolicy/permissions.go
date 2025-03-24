// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package permissionspolicy provides simple tools to create and modify a Permissions Policy.
package permissionspolicy

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// Policy values.
const (
	HeaderName = "Permissions-Policy"

	Accelerometer              = "accelerometer"
	AmbientLightSensor         = "ambient-light-sensor"
	AttributionReporting       = "attribution-reporting"
	Autoplay                   = "autoplay"
	Bluetooth                  = "bluetooth"
	BrowsingTopics             = "browsing-topics"
	Camera                     = "camera"
	ComputePressure            = "compute-pressure"
	CrossOriginIsolated        = "cross-origin-isolated"
	DisplayCapture             = "display-capture"
	DocumentDomain             = "document-domain"
	EncryptedMedia             = "encrypted-media"
	Fullscreen                 = "fullscreen"
	Geolocation                = "geolocation"
	Gyroscope                  = "gyroscope"
	Hid                        = "hid"
	IdentityCredentialsGet     = "identity-credentials-get"
	IdleDetection              = "idle-detection"
	InterestCohort             = "interest-cohort"
	LocalFonts                 = "local-fonts"
	Magnetometer               = "magnetometer"
	Microphone                 = "microphone"
	Midi                       = "midi"
	OtpCredentials             = "otp-credentials"
	Payment                    = "payment"
	PictureInPicture           = "picture-in-picture"
	PublickeyCredentialsCreate = "publickey-credentials-create"
	PublickeyCredentialsGet    = "publickey-credentials-get"
	ScreenWakeLock             = "screen-wake-lock"
	Serial                     = "serial"
	StorageAccess              = "storage-access"
	Usb                        = "usb"
	WebShare                   = "web-share"
	Wildcards                  = "wildcards"
	WindowManagement           = "window-management"
	XrSpatialTracking          = "xr-spatial-tracking"
)

// Policy is a map of permission directives.
// It's the same data structure as http.Header, with a
// different serialization.
type Policy map[string][]string

// DefaultPolicy is a default [Policy] that denies everything.
var DefaultPolicy = Policy{
	Accelerometer:              []string{},
	AmbientLightSensor:         []string{},
	AttributionReporting:       []string{},
	Autoplay:                   []string{},
	Bluetooth:                  []string{},
	BrowsingTopics:             []string{},
	Camera:                     []string{},
	ComputePressure:            []string{},
	CrossOriginIsolated:        []string{},
	DisplayCapture:             []string{},
	DocumentDomain:             []string{},
	EncryptedMedia:             []string{},
	Fullscreen:                 []string{},
	Geolocation:                []string{},
	Gyroscope:                  []string{},
	Hid:                        []string{},
	IdentityCredentialsGet:     []string{},
	IdleDetection:              []string{},
	InterestCohort:             []string{},
	LocalFonts:                 []string{},
	Magnetometer:               []string{},
	Microphone:                 []string{},
	Midi:                       []string{},
	OtpCredentials:             []string{},
	Payment:                    []string{},
	PictureInPicture:           []string{},
	PublickeyCredentialsCreate: []string{},
	PublickeyCredentialsGet:    []string{},
	ScreenWakeLock:             []string{},
	Serial:                     []string{},
	StorageAccess:              []string{},
	Usb:                        []string{},
	WebShare:                   []string{},
	Wildcards:                  []string{},
	WindowManagement:           []string{},
	XrSpatialTracking:          []string{},
}

// Add adds values to an existing directive, or creates it
// if it does not exist.
func (p Policy) Add(name string, values ...string) {
	p[name] = append(p[name], values...)
}

// Set creates or replaces a directive.
func (p Policy) Set(name string, values ...string) {
	p[name] = values
}

// String returns the policy suitable for an http.Header value.
func (p Policy) String() string {
	keys := []string{}
	for k := range p {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	w := &strings.Builder{}
	for _, k := range keys {
		fmt.Fprintf(w, "%s=(%s), ", k, strings.Join(p[k], " "))
	}

	return strings.TrimRight(w.String(), ", ")
}

// Write sets the CSP header to an http.Header.
func (p Policy) Write(h http.Header) {
	h.Set(HeaderName, p.String())
}
