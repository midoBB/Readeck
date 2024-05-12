// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package users

import "slices"

// This file contains the class inventory of user defined display options

var readerFontList = [][2]string{
	{"lora", "font-lora underline-offset-[3px]"},
	{"public-sans", "font-public-sans"},
	{"merriweather", "font-merriweather underline-offset-[3px]"},
	{"inter", "font-inter"},
	{"plex-serif", "font-plex-serif underline-offset-[3px]"},
	{"luciole", "font-luciole decoration-1"},
	{"atkinson-hyperlegible", "font-atkinson-hyperlegible underline-offset-[3px]"},
	{"jetbrains-mono", "font-jetbrains-mono"},
}

var readerFontSizes = []string{
	"text-sm", "text-base", "text-lg",
	"text-xl", "text-2xl", "text-3xl",
}

var readerFontLineHeights = []string{
	"leading-[1.1]", "leading-tight", "leading-normal",
	"leading-relaxed", "leading-loose", "leading-[2.25]",
}

// FontList returns the available fonts.
func (rs *ReaderSettings) FontList() [][2]string {
	return readerFontList
}

// FontSizes returns the available font sizes.
func (rs *ReaderSettings) FontSizes() []string {
	return readerFontSizes
}

// LineHeights returns the available line heights.
func (rs *ReaderSettings) LineHeights() []string {
	return readerFontLineHeights
}

// FontClass returns the font CSS class(es).
func (rs *ReaderSettings) FontClass() string {
	if idx := slices.IndexFunc(readerFontList, func(e [2]string) bool {
		return e[0] == rs.Font
	}); idx >= 0 {
		return readerFontList[idx][1]
	}
	return readerFontList[0][1]
}

// FontSizeClass returns the font size CSS class.
func (rs *ReaderSettings) FontSizeClass() string {
	if idx := rs.FontSize - 1; idx >= 0 && idx < len(readerFontSizes) {
		return readerFontSizes[idx]
	}
	return readerFontSizes[2]
}

// LineHeightClass returns the line height CSS class.
func (rs *ReaderSettings) LineHeightClass() string {
	if idx := rs.LineHeight - 1; idx >= 0 && idx < len(readerFontLineHeights) {
		return readerFontLineHeights[idx]
	}
	return readerFontLineHeights[2]
}
