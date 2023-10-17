package mailgunmailer

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/mailgun/mailgun-go/v4"
	"github.com/stephenafamo/mailer"
)

// mailgunImpl is an instance of the mailgunImpl mailer
type mailgunImpl struct {
	*mailgun.MailgunImpl
	name string
}

// New creates an instance of the Mailgun mailer
// domain (string): Mailgun domain
// apiKey (string): Mailgun API Key
func New(name string, impl *mailgun.MailgunImpl) *mailgunImpl {
	return &mailgunImpl{
		MailgunImpl: impl,
		name:        name,
	}
}

// Send satisfies the mailer interface
func (m mailgunImpl) Send(ctx context.Context, email mailer.Email) (string, string, error) {
	sender := email.FromName + "<" + email.From + ">"

	message := m.NewMessage(sender, email.Subject, email.TextBody)
	message.SetHtml(email.HTMLBody)

	toNLen := len(email.ToNames) > 0
	for k, v := range email.To {
		var err error
		if toNLen {
			err = message.AddRecipient(email.ToNames[k] + "<" + v + ">")
		} else {
			err = message.AddRecipient(v)
		}
		if err != nil {
			return "", "", fmt.Errorf("adding recipient %q: %v", k, err)
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
			// Add the attachment inline
			message.AddReaderInline(
				attachment.Filename,
				io.NopCloser(bytes.NewBuffer(attachment.Data)),
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

	return m.name, id, err
}
