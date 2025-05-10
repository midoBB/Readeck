// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package templates

import (
	"encoding/base64"
	"errors"
	"fmt"
	"image/color"
	"reflect"

	"github.com/CloudyKit/jet/v6"
	"github.com/skip2/go-qrcode"

	"codeberg.org/readeck/readeck/pkg/libjet"
)

func renderQRCode(args jet.Arguments) reflect.Value {
	args.RequireNumOfArguments("qrcode", 1, 3)
	value := args.Get(0).String()
	size := 240
	clr := "#000000"
	if args.NumOfArguments() > 1 {
		size = libjet.ToInt[int](args.Get(1))
	}

	if args.NumOfArguments() > 2 {
		clr = libjet.ToString(args.Get(2))
	}

	qr, err := qrcode.New(value, qrcode.Medium)
	if err != nil {
		panic(err)
	}
	qr.ForegroundColor, _ = parseHexColor(clr)
	qr.DisableBorder = true
	buf, err := qr.PNG(size)
	if err != nil {
		panic(err)
	}

	return reflect.ValueOf("data:image/png;base64," + base64.StdEncoding.EncodeToString(buf))
}

func parseHexColor(s string) (c color.RGBA, err error) {
	c.A = 255
	switch len(s) {
	case 7:
		_, err = fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	case 4:
		_, err = fmt.Sscanf(s, "#%01x%01x%01x", &c.R, &c.G, &c.B)
		c.R *= 17
		c.G *= 17
		c.B *= 17
	default:
		err = errors.New("invalid length")
	}
	return
}
