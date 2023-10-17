package mailer

import (
	"context"
	"errors"
	"fmt"
)

// Mailer sends emails
type Mailer interface {
	Send(ctx context.Context, email Email) (provider, msdID string, err error)
}

// MultiMailer takes multiple mailers and sends with them in sucession
// This is to be used for failovers
type MultiMailer struct {
	mailers []Mailer
}

// NewMultiMailer creates a new instance of a MultiMailer from a bunch of other mailers
func NewMultiMailer(mailers ...Mailer) (*MultiMailer, error) {
	if len(mailers) < 1 {
		return nil, errors.New("At least one Mailer should be given")
	}
	return &MultiMailer{
		mailers: mailers,
	}, nil
}

// Send satisfies the Mailer interface.
// If an error is returned from this, it means ALL the mailers failed
// and the error will be a combination of all the errors
// received from every component mailer
// However, if any Mailer is successful,
// it will supress the errors from the previous ones
func (m MultiMailer) Send(ctx context.Context, email Email) (string, string, error) {
	var err error

	for key, mailer := range m.mailers {
		provider, msgID, err2 := mailer.Send(ctx, email)
		if err2 != nil {
			if err == nil {
				err = fmt.Errorf("%d: %v", key, err2)
			} else {
				err = fmt.Errorf("%d: %v\n%v", key, err2, err)
			}
			continue
		}

		// was successful
		return provider, msgID, nil
	}
	return "", "", err
}

// Email all the things. The ToNames and friends are parallel arrays and must
// be 0-length or the same length as their counterpart. To omit a name
// for a user at an index in To simply use an empty string at that
// index in ToNames.
type Email struct {
	To, Cc, Bcc                []string
	ToNames, CcNames, BccNames []string
	FromName, From             string
	ReplyToName, ReplyTo       string
	Subject                    string
	Attachments                []Attachment

	TextBody string
	HTMLBody string
}

// Attachment represents an email attachment
type Attachment struct {
	Filename string
	Data     []byte
	Inline   bool
}
