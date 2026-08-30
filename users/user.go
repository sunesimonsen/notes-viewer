package users

import (
	"net/mail"
	"time"
)

type User struct {
	VerificationCodeHash      string
	VerificationCodeExpiresAt time.Time
	VerificationCodeSentAt    time.Time
	VerificationAttempts      int
	IsVerified                bool
	Email                     mail.Address
}

func NewUser(email mail.Address) User {
	return User{Email: email}
}

func (u User) IsVerificationRequested() bool {
	return u.VerificationCodeHash != "" && time.Now().Before(u.VerificationCodeExpiresAt)
}

// IsVerified implements the passwords.User interface.
func (u User) Verified() bool {
	return u.IsVerified
}
