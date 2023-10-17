package postmarkmailer

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/mrz1836/postmark"
	"github.com/stephenafamo/mailer"
)

// postmarkImpl is an instance of the postmarkImpl mailer
type postmarkImpl struct {
	*postmark.Client
	name string
}

// New creates an instance of the Mailgun mailer
// domain (string): Mailgun domain
// apiKey (string): Mailgun API Key
func New(name string, impl *postmark.Client) *postmarkImpl {
	return &postmarkImpl{
		Client: impl,
		name:   name,
	}
}

// Send satisfies the mailer interface
func (p postmarkImpl) Send(ctx context.Context, email mailer.Email) (string, string, error) {
	from := email.From
	if email.FromName != "" {
		from = email.FromName + "<" + email.From + ">"
	}

	replyTo := email.ReplyTo
	if email.ReplyToName != "" {
		replyTo = email.ReplyToName + "<" + email.ReplyTo + ">"
	}

	var tos, ccs, bccs []string

	toNLen := len(email.ToNames) > 0
	for k, v := range email.To {
		if toNLen {
			tos = append(tos, v)
		} else {
			tos = append(tos, email.ToNames[k]+" <"+v+">")
		}
	}

	ccNLen := len(email.CcNames) > 0
	for k, v := range email.Cc {
		if ccNLen {
			ccs = append(ccs, v)
		} else {
			ccs = append(ccs, email.CcNames[k]+" <"+v+">")
		}
	}

	bccNLen := len(email.BccNames) > 0
	for k, v := range email.Bcc {
		if bccNLen {
			bccs = append(bccs, v)
		} else {
			bccs = append(bccs, email.BccNames[k]+" <"+v+">")
		}
	}

	message := postmark.Email{
		From:     from,
		ReplyTo:  replyTo,
		To:       strings.Join(tos, ","),
		Cc:       strings.Join(ccs, ","),
		Bcc:      strings.Join(bccs, ","),
		Subject:  email.Subject,
		HTMLBody: email.HTMLBody,
		TextBody: email.TextBody,
	}

	for _, attachment := range email.Attachments {
		b := make([]byte, base64.StdEncoding.EncodedLen(len(attachment.Data)))
		base64.StdEncoding.Encode(b, attachment.Data)

		attach := postmark.Attachment{
			Name:        attachment.Filename,
			Content:     string(b),
			ContentType: http.DetectContentType(attachment.Data),
		}

		if attachment.Inline {
			attach.ContentID = fmt.Sprintf("cid:%s", attachment.Filename)
		}

		message.Attachments = append(message.Attachments, attach)
	}

	resp, err := p.Client.SendEmail(context.Background(), message)
	if err != nil {
		return "", "", err
	}

	return p.name, resp.MessageID, nil
}
