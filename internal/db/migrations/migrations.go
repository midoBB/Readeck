// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

import "embed"

// Files contains all the static files needed by the app
//
//go:embed */*
var Files embed.FS
