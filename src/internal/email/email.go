// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package email

import (
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"

	"codeberg.org/readeck/readeck/assets"
	"codeberg.org/readeck/readeck/configs"
)

// Sender is the default email sender. It's made public so it can be
// overridden during tests.
var Sender sender

// InitSender initializes the default email sender base on the
// configuration.
func InitSender() {
	if configs.Config.Email.Debug {
		Sender = &StdOutSender{}
		return
	}

	if configs.Config.Email.Host != "" {
		Sender = &SMTPSender{}
		return
	}
}

// CanSendEmail returns true when we can send emails.
func CanSendEmail() bool {
	return Sender != nil
}

// SendEmail sends a message to the given email address using
// a template (go template from assets/templates/emails).
func SendEmail(
	from string, to string, subject string,
	tpl string, context map[string]interface{},
) error {
	if !CanSendEmail() {
		return errors.New("no email sender defined")
	}

	m := mail.NewMSG()
	m.SetFrom(from).
		AddTo(to).
		SetSubject(subject)

	t, err := template.ParseFS(assets.TemplatesFS(), path.Join("emails", tpl))
	if err != nil {
		return err
	}

	buf := strings.Builder{}
	if err = t.Execute(&buf, context); err != nil {
		return err
	}

	m.SetBody(mail.TextPlain, buf.String())
	if m.Error != nil {
		return m.Error
	}

	return Sender.SendEmail(m)
}

// sender defines an email sender.
type sender interface {
	SendEmail(*mail.Email) error
}

// StdOutSender implements EmailSender for stdout.
type StdOutSender struct{}

// SendEmail "sends" an email to stdout.
func (s *StdOutSender) SendEmail(m *mail.Email) error {
	fmt.Fprintln(os.Stdout, "=== Outbound email ===================================================")
	fmt.Fprint(os.Stdout, m.GetMessage())
	fmt.Fprintln(os.Stdout, "\n======================================================================")
	return nil
}

// SMTPSender implements EmailSender for SMTP.
type SMTPSender struct{}

// SendEmail sends an email using SMTP.
func (s *SMTPSender) SendEmail(m *mail.Email) error {
	server := mail.NewSMTPClient()
	server.Host = configs.Config.Email.Host
	server.Port = configs.Config.Email.Port
	server.Username = configs.Config.Email.Username
	server.Password = configs.Config.Email.Password

	switch configs.Config.Email.Encryption {
	case "starttls":
		server.Encryption = mail.EncryptionSTARTTLS
	case "ssltls":
		server.Encryption = mail.EncryptionSSLTLS
	}

	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	if configs.Config.Email.Insecure {
		server.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	smtpClient, err := server.Connect()
	if err != nil {
		return err
	}

	return m.Send(smtpClient)
}
