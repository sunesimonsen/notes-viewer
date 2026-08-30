package emails

import (
	"fmt"
	"net/mail"
	"net/smtp"
)

type SmtpEmailer struct {
	password string
	host     string
	port     string
	from     mail.Address
}

func NewSmtpEmailer(host, port, fromAddress, password string) (SmtpEmailer, error) {
	emailer := SmtpEmailer{
		password: password,
		host:     host,
		port:     port,
	}

	from, err := mail.ParseAddress(fromAddress)
	if err != nil {
		return emailer, fmt.Errorf("parsing SMTP from address: %w", err)
	}

	emailer.from = *from

	return emailer, nil
}

func (e SmtpEmailer) SendMail(to mail.Address, subject string, body string) error {
	// Set up authentication information.
	auth := smtp.PlainAuth("", e.from.Address, e.password, e.host)

	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	msg := []byte(
		"To: " + to.String() + "\r\n" +
			"From: " + e.from.String() + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" +
			body + "\r\n",
	)

	return smtp.SendMail(
		e.host+":"+e.port,
		auth,
		e.from.Address,
		[]string{to.Address},
		msg,
	)
}
