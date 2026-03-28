package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost    = 12
	SessionTTL    = 30 * 24 * time.Hour
	resetTokenTTL = 1 * time.Hour
)

type contextKey int

const contextKeyCustomerID contextKey = iota

// HashPassword hashes plain with bcrypt cost 12.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(b), nil
}

// CheckPassword returns true when plain matches the stored bcrypt hash.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// CreateSession inserts a new session row and returns the random token.
func CreateSession(db *sql.DB, customerID int64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("rand token: %w", err)
	}
	token := hex.EncodeToString(raw)
	expiresAt := time.Now().UTC().Add(SessionTTL).Format(time.RFC3339)

	_, err := db.Exec(
		`INSERT INTO sessions (token, customer_id, expires_at) VALUES (?, ?, ?)`,
		token, customerID, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	return token, nil
}

// GetSession looks up a session token and returns the associated customer ID.
func GetSession(db *sql.DB, token string) (int64, error) {
	var customerID int64
	var expiresAt string

	err := db.QueryRow(
		`SELECT customer_id, expires_at FROM sessions WHERE token = ?`,
		token,
	).Scan(&customerID, &expiresAt)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("session not found")
	}
	if err != nil {
		return 0, fmt.Errorf("query session: %w", err)
	}

	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0, fmt.Errorf("parse expires_at: %w", err)
	}

	if time.Now().UTC().After(exp) {
		_, _ = db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
		return 0, fmt.Errorf("session expired")
	}

	return customerID, nil
}

// DeleteSession removes a session from the database.
func DeleteSession(db *sql.DB, token string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// CreateResetToken inserts a new password_resets row and returns the token.
func CreateResetToken(db *sql.DB, customerID int64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("rand token: %w", err)
	}
	token := hex.EncodeToString(raw)
	expiresAt := time.Now().UTC().Add(resetTokenTTL).Format(time.RFC3339)

	_, err := db.Exec(
		`INSERT INTO password_resets (token, customer_id, expires_at) VALUES (?, ?, ?)`,
		token, customerID, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("insert reset token: %w", err)
	}
	return token, nil
}

// ValidateResetToken checks the token is valid and unused, returns customer ID.
func ValidateResetToken(db *sql.DB, token string) (int64, error) {
	var customerID int64
	var expiresAt string
	var used int

	err := db.QueryRow(
		`SELECT customer_id, expires_at, used FROM password_resets WHERE token = ?`,
		token,
	).Scan(&customerID, &expiresAt, &used)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("reset token not found")
	}
	if err != nil {
		return 0, fmt.Errorf("query reset token: %w", err)
	}

	if used != 0 {
		return 0, fmt.Errorf("reset token already used")
	}

	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0, fmt.Errorf("parse expires_at: %w", err)
	}

	if time.Now().UTC().After(exp) {
		return 0, fmt.Errorf("reset token expired")
	}

	return customerID, nil
}

// ConsumeResetToken marks the token as used.
func ConsumeResetToken(db *sql.DB, token string) error {
	_, err := db.Exec(`UPDATE password_resets SET used = 1 WHERE token = ?`, token)
	if err != nil {
		return fmt.Errorf("consume reset token: %w", err)
	}
	return nil
}

// Middleware returns HTTP middleware that validates the session cookie and
// injects the customer ID into the request context.
func Middleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			customerID, err := GetSession(db, cookie.Value)
			if err != nil {
				http.SetCookie(w, &http.Cookie{
					Name:     "session",
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyCustomerID, customerID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CustomerIDFromContext extracts the customer ID stored by Middleware.
func CustomerIDFromContext(ctx context.Context) int64 {
	v, _ := ctx.Value(contextKeyCustomerID).(int64)
	return v
}
