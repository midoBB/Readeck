// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"

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

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	serverURL := fmt.Sprintf(
		"http://%s:%d/%s",
		configs.Config.Server.Host,
		configs.Config.Server.Port,
		configs.Config.Server.Prefix,
	)

	println("Checking URL:", serverURL)
	rsp, err := client.Get(serverURL)
	if err != nil {
		return errors.Join(err, errors.New("Readeck server does not answer"))
	}
	defer rsp.Body.Close() //nolint:errcheck

	if rsp.StatusCode != http.StatusSeeOther {
		return fmt.Errorf(`Invalid response: status %s`, rsp.Status)
	}

	println("All good!")
	return nil
}
