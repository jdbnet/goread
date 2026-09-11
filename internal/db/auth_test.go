package db

import (
	"strings"
	"testing"
	"time"
)

func TestHashPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, passwordHashPrefix+"$") {
		t.Fatalf("unexpected hash format %q", hash)
	}
	if !VerifyPassword(hash, "correct horse") {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword(hash, "wrong password") {
		t.Fatal("wrong password should not verify")
	}
	if VerifyPassword("", "correct horse") {
		t.Fatal("empty hash should not verify")
	}
}

func TestCredentialsAndSessions(t *testing.T) {
	d, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	enabled, err := d.AuthEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("auth should be off on a new database")
	}

	if err := d.SetCredentials("jamie", "secret12"); err != nil {
		t.Fatal(err)
	}
	enabled, err = d.AuthEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("auth should be on after SetCredentials")
	}
	c, err := d.GetAuth()
	if err != nil {
		t.Fatal(err)
	}
	if c.Username != "jamie" {
		t.Fatalf("username %q", c.Username)
	}
	if c.PasswordHash == "secret12" {
		t.Fatal("password must be hashed")
	}
	if !VerifyPassword(c.PasswordHash, "secret12") {
		t.Fatal("stored hash should match password")
	}

	token, _, err := d.CreateSession(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := d.ValidSession(token)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("fresh session should be valid")
	}
	expired, _, err := d.CreateSession(-time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ok, err = d.ValidSession(expired)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expired session should be invalid")
	}

	if err := d.ClearCredentials(); err != nil {
		t.Fatal(err)
	}
	enabled, err = d.AuthEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("auth should be off after ClearCredentials")
	}
}

func TestValidateUsername(t *testing.T) {
	if err := ValidateUsername(""); err == nil {
		t.Fatal("empty username should fail")
	}
	if err := ValidateUsername(strings.Repeat("a", 65)); err == nil {
		t.Fatal("long username should fail")
	}
	if err := ValidateUsername("ok"); err != nil {
		t.Fatal(err)
	}
}
