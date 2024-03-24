// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"

	"github.com/cristalhq/acmd"

	"codeberg.org/readeck/readeck/configs"
)

func init() {
	commands = append(commands, acmd.Command{
		Name:        "healthcheck",
		Description: "Check if the server is running",
		ExecFunc:    runHealthcheck,
	})
}

func runHealthcheck(_ context.Context, args []string) error {
	var flags appFlags
	if err := flags.Flags().Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if flags.ConfigFile == "" {
		flags.ConfigFile = "config.toml"
	}

	if err := configs.LoadConfiguration(flags.ConfigFile); err != nil {
		return fmt.Errorf("error loading configuration (%s)", err)
	}

	// Try to open a TCP connection to the HTTP server.
	conn, err := net.Dial("tcp", fmt.Sprintf(
		"%s:%d",
		configs.Config.Server.Host,
		configs.Config.Server.Port,
	))
	if err != nil {
		return errors.Join(err, errors.New("Readeck server does not answer"))
	}
	defer conn.Close() //nolint:errcheck

	fmt.Printf("Readeck is listening on %s\n", conn.RemoteAddr())
	return nil
}
