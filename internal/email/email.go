package email

import (
	"crypto/tls"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"

	"github.com/readeck/readeck/assets"
	"github.com/readeck/readeck/configs"
)

// SendEmail sends a message to the given email address using
// a template (go template from assets/templates/emails).
func SendEmail(
	from string, to string, subject string,
	tpl string, context map[string]interface{},
) error {
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

	if configs.Config.Email.Debug {
		return sendStdout(m)
	}
	return sendSMTP(m)
}

// sendStdout "sends" an email to stdout.
func sendStdout(m *mail.Email) error {
	fmt.Fprintln(os.Stdout, "=== Outbound email ===================================================")
	fmt.Fprint(os.Stdout, m.GetMessage())
	fmt.Fprintln(os.Stdout, "\n======================================================================")
	return nil
}

// sendSMTP sends the message using smtp.
func sendSMTP(m *mail.Email) error {
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
