package users

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"
)

type TokenGenerator interface {
	Generate() (string, error)
}

// RandomTokenGenerator generates random 6-digit verification codes.
type RandomTokenGenerator struct{}

func (g RandomTokenGenerator) Generate() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

type Emailer interface {
	SendMail(to mail.Address, subject string, body string) error
}

type Verification struct {
	Tokens  TokenGenerator
	Emailer Emailer
}

const subject = "Pass viewer authentication"

const (
	verificationCodeTTL        = 10 * time.Minute
	verificationMaxTrials      = 5
	verificationResendCooldown = 1 * time.Minute
)

func (s Verification) SendCode(user *User) error {
	code, err := s.Tokens.Generate()
	if err != nil {
		return err
	}

	user.VerificationCodeHash = hashToken(code)
	user.VerificationCodeExpiresAt = time.Now().Add(verificationCodeTTL)
	user.VerificationCodeSentAt = time.Now()
	user.VerificationAttempts = 0

	message := strings.Join([]string{
		"Verification code",
		"",
		code,
	}, "\n")

	err = s.Emailer.SendMail(user.Email, subject, message)
	if err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	return nil
}

func (s Verification) VerifyCode(user *User, code string) {
	if user.IsVerified {
		return
	}

	now := time.Now()
	if user.VerificationCodeHash == "" || now.After(user.VerificationCodeExpiresAt) {
		user.VerificationCodeHash = ""
		user.VerificationAttempts = 0
		return
	}

	user.VerificationAttempts++
	if user.VerificationAttempts >= verificationMaxTrials {
		user.VerificationAttempts = 0
		user.VerificationCodeHash = ""
		return
	}

	if tokensMatch(user.VerificationCodeHash, code) {
		user.IsVerified = true
		user.VerificationCodeHash = ""
		user.VerificationAttempts = 0
	}
}

func (s Verification) CanSend(user *User, now time.Time) (bool, time.Duration) {
	if user.VerificationCodeSentAt.IsZero() {
		return true, 0
	}
	remaining := verificationResendCooldown - now.Sub(user.VerificationCodeSentAt)
	if remaining <= 0 {
		return true, 0
	}
	return false, remaining
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func tokensMatch(expectedHash string, token string) bool {
	actualHash := hashToken(token)
	return subtle.ConstantTimeCompare([]byte(expectedHash), []byte(actualHash)) == 1
}
