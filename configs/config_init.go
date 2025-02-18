// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package configs

import (
	crand "crypto/rand"
	"math/rand/v2"
	"os"
	"text/template"
)

const initialConfiguration = `
[main]
log_level = "{{ .Main.LogLevel }}"
secret_key = "{{ .Main.SecretKey }}"
data_directory = "{{ .Main.DataDirectory }}"

[server]
host = "{{ .Server.Host }}"
port = {{ .Server.Port }}

[database]
source = "{{ .Database.Source }}"
`

var keyChars = [][2]rune{
	{0x30, 0x39}, {0x41, 0x54}, {0x61, 0x7a}, // latin alphabet
	{0x21, 0x21}, {0x23, 0x26}, {0x2a, 0x2b}, // symbols
}

// WriteConfig writes configuration to a file.
func WriteConfig(filename string) error {
	tmpl, err := template.New("cfg").Parse(initialConfiguration)
	if err != nil {
		return err
	}

	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	if err = tmpl.Execute(fd, Config); err != nil {
		defer fd.Close() //nolint:errcheck
		return err
	}

	return fd.Close()
}

// GenerateKey returns a random key.
func GenerateKey(minLen, maxLen int) string {
	if minLen >= maxLen {
		panic("maxLen must be greater then minLen")
	}

	runes := []rune{}
	for _, table := range keyChars {
		for i := table[0]; i <= table[1]; i++ {
			runes = append(runes, i)
		}
	}

	l := rand.N(maxLen-minLen) + minLen //nolint:gosec
	b := make([]byte, l)
	if _, err := crand.Read(b); err != nil {
		panic(err)
	}
	for i := 0; i < l; i++ {
		b[i] = byte(runes[b[i]%byte(len(runes))])
	}

	return string(b)
}
