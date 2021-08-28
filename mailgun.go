package mailer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"

	"github.com/mailgun/mailgun-go/v4"
)

// Mailgun is an instance of the Mailgun mailer
type Mailgun struct {
	*mailgun.MailgunImpl
	Name string
}

// NewMailgun creates an instance of the Mailgun mailer
// domain (string): Mailgun domain
// apiKey (string): Mailgun API Key
func NewMailgun(name string, impl *mailgun.MailgunImpl) *Mailgun {
	return &Mailgun{
		MailgunImpl: impl,
		Name:        name,
	}
}

// Send satisfies the mailer interface
func (m Mailgun) Send(ctx context.Context, email Email) (string, string, error) {
	sender := email.FromName + "<" + email.From + ">"

	message := m.NewMessage(sender, email.Subject, email.TextBody)
	message.SetHtml(email.HTMLBody)

	toNLen := len(email.ToNames) > 0
	for k, v := range email.To {
		if toNLen {
			message.AddRecipient(email.ToNames[k] + "<" + v + ">")
		} else {
			message.AddRecipient(v)
		}
	}

	ccNLen := len(email.CcNames) > 0
	for k, v := range email.Cc {
		if ccNLen {
			message.AddCC(email.CcNames[k] + "<" + v + ">")
		} else {
			message.AddCC(v)
		}
	}

	bccNLen := len(email.BccNames) > 0
	for k, v := range email.Bcc {
		if bccNLen {
			message.AddBCC(email.BccNames[k] + "<" + v + ">")
		} else {
			message.AddBCC(v)
		}
	}

	for _, attachment := range email.Attachments {
		if attachment.Inline {
			// Decode the base64 attachment
			b := make([]byte, base64.StdEncoding.DecodedLen(len(attachment.Base64)))
			_, err := base64.StdEncoding.Decode(b, attachment.Base64)
			if err != nil {
				return "", "", fmt.Errorf("Error decoding base64 attachment: %v", err)
			}

			// Add the attachment inline
			message.AddReaderInline(
				attachment.Filename,
				ioutil.NopCloser(bytes.NewBuffer(b)),
			)
		} else {
			message.AddBufferAttachment(
				attachment.Filename,
				attachment.Data,
			)
		}
	}

	if email.ReplyTo != "" {
		message.SetReplyTo(email.ReplyToName + "<" + email.ReplyTo + ">")
	}

	_, id, err := m.MailgunImpl.Send(context.Background(), message)

	return m.Name, id, err
}
