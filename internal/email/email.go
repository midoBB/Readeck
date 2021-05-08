package email

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"text/template"

	"gopkg.in/gomail.v2"

	"github.com/readeck/readeck/assets"
	"github.com/readeck/readeck/configs"
)

// SendEmail sends a message to the given email address using
// a template (go template from assets/templates/emails).
func SendEmail(
	from string, to string, subject string,
	tpl string, context map[string]interface{},
) error {
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)

	t, err := template.ParseFS(assets.TemplatesFS(), path.Join("emails", tpl))
	if err != nil {
		return err
	}

	buf := strings.Builder{}
	if err = t.Execute(&buf, context); err != nil {
		return err
	}
	m.SetBody("text/plain", buf.String())

	if configs.Config.Email.Debug {
		return sendStdout(m)
	}
	return sendSMTP(m)
}

// sendStdout "sends" an email to stdout.
func sendStdout(m *gomail.Message) error {
	s := gomail.SendFunc(stdoutSender)
	return gomail.Send(s, m)
}

func stdoutSender(from string, to []string, msg io.WriterTo) error {
	fmt.Fprintln(os.Stdout, "=== Outbound email ===================================================")
	fmt.Fprintf(os.Stdout, "From: %s\n", from)
	fmt.Fprintf(os.Stdout, "To: %#v\n\n~~\n", to)
	msg.WriteTo(os.Stdout)
	fmt.Fprintln(os.Stdout, "\n======================================================================")
	return nil
}

// sendSMTP sends the message using smtp.
func sendSMTP(m *gomail.Message) error {
	d := gomail.Dialer{
		Host:     configs.Config.Email.Host,
		Port:     configs.Config.Email.Port,
		Username: configs.Config.Email.Username,
		Password: configs.Config.Email.Password,
		SSL:      configs.Config.Email.SSL,
	}

	s, err := d.Dial()
	if err != nil {
		return err
	}

	return gomail.Send(s, m)
}
