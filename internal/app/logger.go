// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"log/slog"

	. "github.com/phsym/console-slog" //nolint:revive
)

type devLogTheme struct{}

func (t devLogTheme) Name() string            { return "" }
func (t devLogTheme) Timestamp() ANSIMod      { return ToANSICode(BrightBlack) }
func (t devLogTheme) Source() ANSIMod         { return ToANSICode(Bold, BrightBlack) }
func (t devLogTheme) Message() ANSIMod        { return ToANSICode(Bold) }
func (t devLogTheme) MessageDebug() ANSIMod   { return ToANSICode() }
func (t devLogTheme) AttrKey() ANSIMod        { return ToANSICode(Cyan) }
func (t devLogTheme) AttrValue() ANSIMod      { return ToANSICode(Faint) }
func (t devLogTheme) AttrValueError() ANSIMod { return ToANSICode(Bold, Red) }
func (t devLogTheme) LevelError() ANSIMod     { return ToANSICode(Bold, Red) }
func (t devLogTheme) LevelWarn() ANSIMod      { return ToANSICode(Bold, Yellow) }
func (t devLogTheme) LevelInfo() ANSIMod      { return ToANSICode(Bold, Green) }
func (t devLogTheme) LevelDebug() ANSIMod     { return ToANSICode(Bold, BrightMagenta) }
func (t devLogTheme) Level(level slog.Level) ANSIMod {
	switch {
	case level >= slog.LevelError:
		return t.LevelError()
	case level >= slog.LevelWarn:
		return t.LevelWarn()
	case level >= slog.LevelInfo:
		return t.LevelInfo()
	default:
		return t.LevelDebug()
	}
}
