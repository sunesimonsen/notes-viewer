package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"PORT",
		"NOTES_VIEWER_STORE_PATH",
		"NOTES_VIEWER_SMTP_HOST",
		"NOTES_VIEWER_SMTP_PORT",
		"NOTES_VIEWER_SMTP_FROM_ADDRESS",
		"NOTES_VIEWER_SMTP_PASSWORD",
		"NOTES_VIEWER_USER_NAME",
		"NOTES_VIEWER_USER_EMAIL",
		"NOTES_VIEWER_SKIP_VERIFICATION",
	} {
		t.Setenv(key, "")
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Run("returns error when NOTES_VIEWER_STORE_PATH is not set", func(t *testing.T) {
		clearEnv(t)

		_, err := ConfigFromEnv()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "NOTES_VIEWER_STORE_PATH")
	})

	t.Run("defaults port to 8081", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("NOTES_VIEWER_STORE_PATH", "/tmp/store")

		config, err := ConfigFromEnv()
		assert.NoError(t, err)
		assert.Equal(t, "8081", config.Port)
	})

	t.Run("uses provided port", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("PORT", "3000")
		t.Setenv("NOTES_VIEWER_STORE_PATH", "/tmp/store")

		config, err := ConfigFromEnv()
		assert.NoError(t, err)
		assert.Equal(t, "3000", config.Port)
	})

	t.Run("reads all config values", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("PORT", "9090")
		t.Setenv("NOTES_VIEWER_STORE_PATH", "/tmp/store")
		t.Setenv("NOTES_VIEWER_SMTP_HOST", "smtp.test.com")
		t.Setenv("NOTES_VIEWER_SMTP_PORT", "587")
		t.Setenv("NOTES_VIEWER_SMTP_FROM_ADDRESS", "from@test.com")
		t.Setenv("NOTES_VIEWER_SMTP_PASSWORD", "secret")
		t.Setenv("NOTES_VIEWER_USER_NAME", "Test")
		t.Setenv("NOTES_VIEWER_USER_EMAIL", "test@test.com")

		config, err := ConfigFromEnv()
		assert.NoError(t, err)
		assert.Equal(t, "9090", config.Port)
		assert.Equal(t, "/tmp/store", config.NotesStorePath)
		assert.Equal(t, "smtp.test.com", config.SMTPHost)
		assert.Equal(t, "587", config.SMTPPort)
		assert.Equal(t, "from@test.com", config.SMTPFromAddress)
		assert.Equal(t, "secret", config.SMTPPassword)
		assert.Equal(t, "Test", config.UserName)
		assert.Equal(t, "test@test.com", config.UserEmail)
		assert.False(t, config.SkipVerification)
	})

	t.Run("parses skip verification flag", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("NOTES_VIEWER_STORE_PATH", "/tmp/store")
		t.Setenv("NOTES_VIEWER_SKIP_VERIFICATION", "true")

		config, err := ConfigFromEnv()
		assert.NoError(t, err)
		assert.True(t, config.SkipVerification)
	})

	t.Run("parses skip verification flag case insensitively", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("NOTES_VIEWER_STORE_PATH", "/tmp/store")
		t.Setenv("NOTES_VIEWER_SKIP_VERIFICATION", "TRUE")

		config, err := ConfigFromEnv()
		assert.NoError(t, err)
		assert.True(t, config.SkipVerification)
	})

	t.Run("resolves ~ in password store path", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("NOTES_VIEWER_STORE_PATH", "~/.notes-store")

		home, _ := os.UserHomeDir()

		config, err := ConfigFromEnv()
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join(home, ".notes-store"), config.NotesStorePath)
	})

	t.Run("returns error when home directory used as store path", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("NOTES_VIEWER_STORE_PATH", "~")

		_, err := ConfigFromEnv()
		assert.EqualError(t, err, "NOTES_VIEWER_STORE_PATH is set to ~")
	})

	t.Run("cleans absolute paths", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("NOTES_VIEWER_STORE_PATH", "/tmp/store/../store")

		config, err := ConfigFromEnv()
		assert.NoError(t, err)
		assert.Equal(t, "/tmp/store", config.NotesStorePath)
	})
}
