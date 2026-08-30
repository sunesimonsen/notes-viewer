package server

import (
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestNewServer(t *testing.T) {
	t.Run("builds server with valid config", func(t *testing.T) {
		config := Config{
			NotesStorePath:   filepath.Clean("../test-notes-store"),
			SkipVerification: true,
			SMTPHost:         "smtp.test.com",
			SMTPPort:         "587",
			SMTPFromAddress:  "from@test.com",
			SMTPPassword:     "secret",
			UserName:         "Test",
			UserEmail:        "test@test.com",
		}

		s, err := NewServer(config)

		assert.NoError(t, err)
		assert.NotEqual(t, nil, s)
		assert.True(t, s.skipVerification)
		assert.Equal(t, config.UserName, s.defaultUser.Name)
		assert.Equal(t, config.UserEmail, s.defaultUser.Address)
		assert.NotEqual(t, nil, s.router)
		assert.NotEqual(t, nil, s.sessionManager)
	})
}
