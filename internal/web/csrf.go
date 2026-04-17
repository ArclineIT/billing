package web

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const csrfCookieName = "_csrf"

// ensureCSRFToken returns the current CSRF token from the cookie, generating
// and setting a new one if the cookie is absent.
func ensureCSRFToken(w http.ResponseWriter, r *http.Request, secure bool) string {
	if cookie, err := r.Cookie(csrfCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	token := hex.EncodeToString(raw)
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
	return token
}

// validateCSRF returns true when the form's csrf_token field matches the
// _csrf cookie. Call after r.ParseForm().
func validateCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	formToken := r.FormValue("csrf_token")
	return formToken != "" && formToken == cookie.Value
}
