package users_test

import (
	"errors"
	"net/mail"
	"regexp"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/sunesimonsen/notes-viewer/internal/testutil"
	"github.com/sunesimonsen/notes-viewer/users"
)

func setup(code string) (users.Verification, *testutil.Emailer) {
	emailer := &testutil.Emailer{}
	verification := testutil.NewVerification(code, emailer)

	return verification, emailer
}

func TestVerification(t *testing.T) {
	t.Run("Code mismatch", func(t *testing.T) {
		email := mail.Address{Address: "test@test.com"}
		code := "Real code"

		verification, emailer := setup(code)

		user := users.NewUser(email)
		assert.False(t, user.IsVerificationRequested(), "Verification requested")
		assert.NoError(t, verification.SendCode(&user))
		assert.True(t, user.IsVerificationRequested(), "Verification requested")

		assert.Equal(t, emailer.To, email, "Email user for verification")
		assert.Contains(t, emailer.Message, code, "Code send for verification")

		verification.VerifyCode(&user, "Other code")

		assert.False(t, user.IsVerified, "User is not verified")
	})

	t.Run("Code matches", func(t *testing.T) {
		email := mail.Address{Address: "test@test.com"}
		code := "Real code"

		verification, emailer := setup(code)

		user := users.NewUser(email)
		assert.False(t, user.IsVerificationRequested(), "Verification requested")
		assert.NoError(t, verification.SendCode(&user), "Send code")
		assert.True(t, user.IsVerificationRequested(), "Verification requested")

		assert.Equal(t, emailer.To, email, "Email user for verification")
		assert.Contains(t, emailer.Message, code, "Code send for verification")

		verification.VerifyCode(&user, "Real code")

		assert.True(t, user.IsVerified, "User is verified")
	})

	t.Run("Sending email fails", func(t *testing.T) {
		email := mail.Address{Address: "test@test.com"}
		code := "Real code"

		verification, emailer := setup(code)
		emailer.Err = errors.New("Testing")

		user := users.NewUser(email)
		assert.Error(t, verification.SendCode(&user), "Send code")
	})

	t.Run("Resets code after 5 attempts", func(t *testing.T) {
		email := mail.Address{Address: "test@test.com"}
		code := "Real code"

		verification, _ := setup(code)

		user := users.NewUser(email)
		assert.NoError(t, verification.SendCode(&user), "Send code")

		for range 4 {
			verification.VerifyCode(&user, "Wrong code")
		}

		assert.Equal(t, 4, user.VerificationAttempts, "Attempts increment before reset")
		assert.NotEqual(t, "", user.VerificationCodeHash, "Code hash still set before reset")

		verification.VerifyCode(&user, "Wrong code")

		assert.Equal(t, 0, user.VerificationAttempts, "Attempts reset after limit")
		assert.Equal(t, "", user.VerificationCodeHash, "Code cleared after limit")
	})

	t.Run("Does not verify when code is expired", func(t *testing.T) {
		email := mail.Address{Address: "test@test.com"}
		code := "Real code"

		verification, _ := setup(code)
		user := users.NewUser(email)
		assert.NoError(t, verification.SendCode(&user), "Send code")

		user.VerificationCodeExpiresAt = time.Now().Add(-1 * time.Minute)
		user.VerificationAttempts = 2

		verification.VerifyCode(&user, code)

		assert.False(t, user.IsVerified, "User remains unverified")
		assert.Equal(t, "", user.VerificationCodeHash, "Code cleared after expiry")
		assert.Equal(t, 0, user.VerificationAttempts, "Attempts reset after expiry")
	})

	t.Run("Does not mutate verified users", func(t *testing.T) {
		email := mail.Address{Address: "test@test.com"}
		verification, _ := setup("Real code")
		user := users.NewUser(email)
		user.IsVerified = true
		user.VerificationCodeHash = "hash"
		user.VerificationAttempts = 3

		verification.VerifyCode(&user, "Real code")

		assert.True(t, user.IsVerified, "User stays verified")
		assert.Equal(t, "hash", user.VerificationCodeHash, "Code hash remains")
		assert.Equal(t, 3, user.VerificationAttempts, "Attempts remain")
	})
}

func TestVerificationCanSend(t *testing.T) {
	verification, _ := setup("123456")
	now := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)

	t.Run("Allows when no code has been sent", func(t *testing.T) {
		user := users.NewUser(mail.Address{Address: "test@test.com"})

		allowed, remaining := verification.CanSend(&user, now)

		assert.True(t, allowed)
		assert.Equal(t, time.Duration(0), remaining)
	})

	t.Run("Blocks within cooldown window", func(t *testing.T) {
		user := users.NewUser(mail.Address{Address: "test@test.com"})
		user.VerificationCodeSentAt = now.Add(-30 * time.Second)

		allowed, remaining := verification.CanSend(&user, now)

		assert.False(t, allowed)
		assert.Equal(t, 30*time.Second, remaining)
	})

	t.Run("Allows after cooldown window", func(t *testing.T) {
		user := users.NewUser(mail.Address{Address: "test@test.com"})
		user.VerificationCodeSentAt = now.Add(-2 * time.Minute)

		allowed, remaining := verification.CanSend(&user, now)

		assert.True(t, allowed)
		assert.Equal(t, time.Duration(0), remaining)
	})
}

var sixDigits = regexp.MustCompile(`^\d{6}$`)

func TestRandomTokenGenerator(t *testing.T) {
	gen := users.RandomTokenGenerator{}

	t.Run("generates a 6-digit string", func(t *testing.T) {
		token, err := gen.Generate()
		assert.NoError(t, err)
		assert.True(t, sixDigits.MatchString(token), "token should be exactly 6 digits, got: %s", token)
	})

	t.Run("generates different tokens", func(t *testing.T) {
		seen := map[string]bool{}
		for range 100 {
			token, err := gen.Generate()
			assert.NoError(t, err)
			seen[token] = true
		}
		// With 100 random 6-digit tokens, we should see many distinct values.
		assert.True(t, len(seen) > 50, "expected diverse tokens, got %d unique out of 100", len(seen))
	})

	t.Run("zero-pads short numbers", func(t *testing.T) {
		// Generate many tokens and verify all are exactly 6 chars.
		for range 1000 {
			token, err := gen.Generate()
			assert.NoError(t, err)
			assert.Equal(t, 6, len(token), "token length should be 6, got: %s", token)
		}
	})
}
