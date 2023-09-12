// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package onboarding

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"golang.org/x/term"

	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/internal/db"
	"codeberg.org/readeck/readeck/pkg/forms"
)

var welcomeMessage = `
 ╔═════════════════════════════════════════════════════════════════════╗
 ║                                                                     ║
 ║                        Welcome to Readeck!                          ║
 ║                                                                     ║
 ║       This is your first installation and we're delighted           ║
 ║                       to have you on board.                         ║
 ║                                                                     ║
 ╚═════════════════════════════════════════════════════════════════════╝

 You only need to create a first user and you'll be ready to go.

 Please note: Your email address is not collected in any way.
              It is stored only on your installation as a password
              recovery mean.

`

// CLI provides a text interface to create the first user when needed.
func CLI() error {
	// Create the first user if needed
	count, err := db.Q().From("user").Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	println(welcomeMessage)

	stop := make(chan os.Signal, 2)
	done := make(chan bool)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	var email string
	var username string
	var password string

	go func() {
		email = stringPrompt("Your email address:", "", func(s string) error {
			f := forms.NewTextField("")
			f.Set(s)
			return forms.IsEmail(f)
		})

		username = stringPrompt("Choose a username:", strings.Split(email, "@")[0], func(s string) error {
			return nil
		})

		password = passwordPrompt("Choose a password:", func(s string) error {
			if len(s) < 8 {
				return errors.New("password must containt at least 8 characters")
			}
			return nil
		})

		done <- true
	}()

	select {
	case <-stop:
		return errors.New("stopped by user")
	case <-done:
		log.WithField("username", username).Info("creating user")
		err := users.Users.Create(&users.User{
			Username: username,
			Email:    email,
			Password: password,
			Group:    "admin",
		})
		if err != nil {
			return err
		}

		println("\n  Readeck is ready!\n")
	}

	return nil
}

func stringPrompt(label string, defaultValue string, validator func(string) error) string {
	if defaultValue != "" {
		label = fmt.Sprintf("%s [%s]", label, defaultValue)
	}

	for {
		var s string
		r := bufio.NewReader(os.Stdin)
		for {
			fmt.Fprint(os.Stderr, label+" ")
			s, _ = r.ReadString('\n')
			if s != "" {
				break
			}
		}

		s = strings.TrimSpace(s)
		if s == "" {
			s = defaultValue
		}
		if err := validator(s); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "\n")
			return s
		}
	}
}

func passwordPrompt(label string, validator func(string) error) string {
	for {
		var s string
		fmt.Fprint(os.Stderr, label+" ")
		b, _ := term.ReadPassword(int(syscall.Stdin))
		s = string(b)

		fmt.Println()
		if err := validator(s); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "\n")
			return s
		}
	}
}
