package mailer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
)

// SMTP is an instance of the SMTP mailer
type SMTP struct {
	name string // provider name of this mailer
	host string
	port int
	auth smtp.Auth
}

// NewSMTP creates an instance of the SMTP mailer
// host (string): SMTP host
// port (int): SMTP port
// username (string): SMTP username
// password (string): SMTP password
func NewSMTP(name, host string, port int, auth smtp.Auth) *SMTP {
	return &SMTP{
		name: name,
		host: host,
		port: port,
		auth: auth,
	}
}

func (s *SMTP) generateMsgID() (string, error) {
	u, err := uuid.NewV4()

	return "<" + u.String() + "@" + s.host + ">", err
}

// Send satisfies the mailer interface
func (s *SMTP) Send(ctx context.Context, email Email) (string, string, error) {
	if email.TextBody == "" && email.HTMLBody == "" {
		return "", "", fmt.Errorf("Email must have either Text or HTML body")
	}

	messageID, err := s.generateMsgID()
	if err != nil {
		return "", "", fmt.Errorf("Can't generate message ID: %v", err)
	}

	delimeterOuter := "boundary-outer"
	delimeter := "boundary"
	from := email.FromName + "<" + email.From + ">"

	var tos, ccs, bccs []string

	toNLen := len(email.ToNames) > 0
	for k, v := range email.To {
		addr := mail.Address{Address: v}
		if toNLen {
			addr.Name = email.ToNames[k]
		}
		tos = append(tos, addr.String())
	}

	ccNLen := len(email.CcNames) > 0
	for k, v := range email.Cc {
		addr := mail.Address{Address: v}
		if ccNLen {
			addr.Name = email.CcNames[k]
		}
		ccs = append(ccs, addr.String())
	}

	bccNLen := len(email.BccNames) > 0
	for k, v := range email.Bcc {
		addr := mail.Address{Address: v}
		if bccNLen {
			addr.Name = email.BccNames[k]
		}
		bccs = append(bccs, addr.String())
	}

	// basic email headers
	msg := fmt.Sprintf("Message-ID: %s\r\n", messageID)
	msg += fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	msg += fmt.Sprintf("From: %s\r\n", from)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(tos, ";"))
	if len(ccs) > 0 {
		msg += fmt.Sprintf("Cc: %s\r\n", strings.Join(ccs, ";"))
	}
	if email.ReplyTo != "" {
		msg += fmt.Sprintf("Reply-To: %s\r\n", email.ReplyToName+"<"+email.ReplyTo+">")
	}
	msg += fmt.Sprintf("Subject: %s\r\n", email.Subject)

	msg += "MIME-Version: 1.0\r\n"

	// Start outer email body
	msg += fmt.Sprintf("Content-Type: multipart/mixed; boundary=%q\r\n", delimeterOuter)

	msg += fmt.Sprintf("\r\n--%s\r\n", delimeterOuter)

	// Add the text/html body
	msg += fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", delimeter)

	if email.TextBody != "" {
		// place Text message
		msg += fmt.Sprintf("\r\n--%s\r\n", delimeter)
		msg += "Content-Transfer-Encoding: 7bit\r\n"
		msg += "Content-Type: text/plain; charset=\"utf-8\"\r\n"
		msg += fmt.Sprintf("\r\n%s\r\n", email.TextBody)
	}

	if email.HTMLBody != "" {
		// place HTML message
		msg += fmt.Sprintf("\r\n--%s\r\n", delimeter)
		msg += "Content-Transfer-Encoding: 7bit\r\n"
		msg += "Content-Type: text/html; charset=\"utf-8\"\r\n"
		msg += fmt.Sprintf("\r\n%s\r\n", email.HTMLBody)
	}

	// End the text/html body
	msg += fmt.Sprintf("\r\n--%s--\r\n", delimeter)

	buf := bytes.NewBuffer([]byte(msg))

	// Add the attachments
	if len(email.Attachments) > 0 {
		buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", delimeterOuter))

		for _, attachment := range email.Attachments {
			idHeader := fmt.Sprintf("Content-ID: <%s>\r\n", attachment.Filename)
			buf.WriteString(idHeader)

			ext := filepath.Ext(attachment.Filename)
			mimetype := mime.TypeByExtension(ext)
			if mimetype != "" {
				mime := fmt.Sprintf("Content-Type: %s\r\n", mimetype)
				buf.WriteString(mime)
			} else {
				buf.WriteString("Content-Type: application/octet-stream\r\n")
			}
			buf.WriteString("Content-Transfer-Encoding: base64\r\n")

			if !attachment.Inline {
				buf.WriteString("Content-Disposition: attachment; filename=\"=?UTF-8?B?")
				buf.WriteString(base64.StdEncoding.EncodeToString([]byte(attachment.Filename)))
				buf.WriteString("?=\"")
			}

			buf.WriteString("\r\n\r\n")

			b := make([]byte, base64.StdEncoding.EncodedLen(len(attachment.Data)))
			base64.StdEncoding.Encode(b, attachment.Data)

			// write base64 content in lines of up to 76 chars
			for i, l := 0, len(b); i < l; i++ {
				buf.WriteByte(b[i])
				if (i+1)%76 == 0 {
					buf.WriteString("\r\n")
				}
			}
		}
	}

	// end the email
	buf.WriteString(fmt.Sprintf("\r\n--%s--\r\n", delimeterOuter))

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	err = smtp.SendMail(
		addr, s.auth,
		email.From, append(append(tos, ccs...), bccs...),
		buf.Bytes(),
	)

	return s.name, messageID, err
}
