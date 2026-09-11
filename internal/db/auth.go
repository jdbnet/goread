package db

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	passwordHashPrefix = "pbkdf2-sha256"
	passwordIterations = 600000
	passwordKeyLen     = 32
	passwordSaltLen    = 16
	sessionTokenBytes  = 32
)

type AuthCredentials struct {
	Username     string
	PasswordHash string
}

func (d *DB) GetAuth() (AuthCredentials, error) {
	var c AuthCredentials
	err := d.SQL.QueryRow(`SELECT username, password_hash FROM auth_credentials WHERE id = 1`).Scan(&c.Username, &c.PasswordHash)
	if err != nil {
		return AuthCredentials{}, err
	}
	return c, nil
}

func (d *DB) AuthEnabled() (bool, error) {
	c, err := d.GetAuth()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(c.Username) != "" && c.PasswordHash != "", nil
}

func (d *DB) SetCredentials(username, password string) error {
	username = strings.TrimSpace(username)
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if err := validatePassword(password); err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	_, err = d.SQL.Exec(`UPDATE auth_credentials SET username=?, password_hash=? WHERE id=1`, username, hash)
	return err
}

func (d *DB) UpdateUsername(username string) error {
	username = strings.TrimSpace(username)
	if err := ValidateUsername(username); err != nil {
		return err
	}
	_, err := d.SQL.Exec(`UPDATE auth_credentials SET username=? WHERE id=1`, username)
	return err
}

func (d *DB) ClearCredentials() error {
	_, err := d.SQL.Exec(`UPDATE auth_credentials SET username='', password_hash='' WHERE id=1`)
	return err
}

func (d *DB) CreateSession(ttl time.Duration) (string, time.Time, error) {
	if err := d.DeleteExpiredSessions(); err != nil {
		return "", time.Time{}, err
	}
	raw := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token := hex.EncodeToString(raw)
	exp := time.Now().Add(ttl).UTC()
	_, err := d.SQL.Exec(`INSERT INTO sessions (token, expires_at) VALUES (?, ?)`, token, formatTime(exp))
	if err != nil {
		return "", time.Time{}, err
	}
	return token, exp, nil
}

func (d *DB) ValidSession(token string) (bool, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return false, nil
	}
	var expires string
	err := d.SQL.QueryRow(`SELECT expires_at FROM sessions WHERE token = ?`, token).Scan(&expires)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	exp, parseErr := time.Parse(time.RFC3339, expires)
	if parseErr != nil {
		_ = d.DeleteSession(token)
		return false, nil
	}
	if time.Now().UTC().After(exp) {
		_ = d.DeleteSession(token)
		return false, nil
	}
	return true, nil
}

func (d *DB) DeleteSession(token string) error {
	_, err := d.SQL.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func (d *DB) DeleteAllSessions() error {
	_, err := d.SQL.Exec(`DELETE FROM sessions`)
	return err
}

func (d *DB) DeleteExpiredSessions() error {
	_, err := d.SQL.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, formatTime(time.Now().UTC()))
	return err
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, passwordSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, passwordKeyLen)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s$%d$%s$%s",
		passwordHashPrefix,
		passwordIterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != passwordHashPrefix {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < 1 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil || len(salt) == 0 {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(want) == 0 {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iter, len(want))
	if err != nil {
		return false
	}
	return hmac.Equal(got, want)
}

func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if len(username) > 64 {
		return fmt.Errorf("username is too long")
	}
	if strings.ContainsAny(username, "\x00\n\r") {
		return fmt.Errorf("username contains invalid characters")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if len(password) > 256 {
		return fmt.Errorf("password is too long")
	}
	return nil
}
