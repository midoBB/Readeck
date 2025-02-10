// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package exp

import (
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// UnicodeCollate is the base Unicode collate.
var UnicodeCollate = collate.New(language.Und, collate.Loose, collate.Numeric)

// UnaccentCompare performs a string comparison after removing accents.
var UnaccentCompare = UnicodeCollate.CompareString
