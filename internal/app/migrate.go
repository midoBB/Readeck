// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"flag"
	"os"

	"github.com/cristalhq/acmd"
)

func init() {
	commands = append(commands, acmd.Command{
		Name:        "migrate",
		Description: "Run database schema migrations",
		ExecFunc:    runMigrate,
	})
}

func runMigrate(_ context.Context, args []string) error {
	var flags appFlags
	if err := flags.Flags().Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Force debug level
	_ = os.Setenv("READECK_DEV_MODE", "1")

	// Init application
	if err := appPreRun(&flags); err != nil {
		return err
	}
	appPostRun()
	return nil
}
