package testutil

import (
	"net/mail"

	"github.com/sunesimonsen/notes-viewer/users"
)

type TokenGenerator struct {
	Code string
}

func (g TokenGenerator) Generate() (string, error) {
	return g.Code, nil
}

type Emailer struct {
	To      mail.Address
	Subject string
	Message string
	Err     error
}

func (e *Emailer) SendMail(to mail.Address, subject string, body string) error {
	e.To = to
	e.Subject = subject
	e.Message = body
	return e.Err
}

func NewVerification(code string, emailer *Emailer) users.Verification {
	return users.Verification{
		Tokens:  TokenGenerator{Code: code},
		Emailer: emailer,
	}
}
