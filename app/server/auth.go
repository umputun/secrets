package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	authCookieName = "secrets_session"
	authUser       = "secrets" // hardcoded username for basic auth
)

// isAuthenticated checks if the request has a valid session cookie
func (s Server) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		return false
	}
	return s.validateSessionToken(cookie.Value)
}

// checkBasicAuth validates basic auth credentials for API access
func (s Server) checkBasicAuth(r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}

	// constant-time username comparison
	usernameCorrect := subtle.ConstantTimeCompare([]byte(username), []byte(authUser)) == 1

	// bcrypt password check (already constant-time)
	passwordCorrect := bcrypt.CompareHashAndPassword([]byte(s.cfg.AuthHash), []byte(password)) == nil

	return usernameCorrect && passwordCorrect
}

// loginCtrl handles login form submission
// POST /login
func (s Server) loginCtrl(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderLoginPopup(w, r, "invalid form data")
		return
	}

	password := r.PostForm.Get("password")

	// validate password against bcrypt hash
	if err := bcrypt.CompareHashAndPassword([]byte(s.cfg.AuthHash), []byte(password)); err != nil {
		s.renderLoginPopup(w, r, "invalid password")
		return
	}

	// authentication successful, set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    s.generateSessionToken(),
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.Protocol == "https",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(s.cfg.SessionTTL.Seconds()),
	})

	// close popup and trigger form resubmit
	w.Header().Set("HX-Trigger", "submitSecretForm")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<div id="popup"></div>`))
}

// logoutCtrl handles logout
// GET /logout
func (s Server) logoutCtrl(w http.ResponseWriter, r *http.Request) {
	// clear the auth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.Protocol == "https",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1, // delete cookie
	})

	// redirect to home
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// loginPopupCtrl returns the login popup HTML
// GET /login-popup
func (s Server) loginPopupCtrl(w http.ResponseWriter, r *http.Request) {
	s.renderLoginPopup(w, r, "")
}

func (s Server) renderLoginPopup(w http.ResponseWriter, r *http.Request, errorMsg string) {
	s.renderLoginPopupWithStatus(w, r, errorMsg, http.StatusOK)
}

func (s Server) renderLoginPopupWithStatus(w http.ResponseWriter, r *http.Request, errorMsg string, status int) {
	data := struct {
		Error string
		Theme string
	}{
		Error: errorMsg,
		Theme: getTheme(r),
	}
	s.render(w, status, "login-popup.tmpl.html", "login-popup", data)
}

// generateSessionToken creates a secure session token
// format: uuid.timestamp.signature
func (s Server) generateSessionToken() string {
	tokenID := uuid.NewString()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// use AuthHash as part of the signing key (derived from signKey in practice)
	secret := s.sessionSecret()

	h := hmac.New(sha256.New, secret)
	h.Write([]byte(tokenID))
	h.Write([]byte(timestamp))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return tokenID + "." + timestamp + "." + signature
}

// validateSessionToken validates a session token
func (s Server) validateSessionToken(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}

	tokenID := parts[0]
	timestamp := parts[1]
	signatureB64 := parts[2]

	// recreate signature
	secret := s.sessionSecret()
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(tokenID))
	h.Write([]byte(timestamp))
	expectedSignature := h.Sum(nil)

	// decode provided signature
	signature, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return false
	}

	// constant-time comparison
	if subtle.ConstantTimeCompare(signature, expectedSignature) != 1 {
		return false
	}

	// check expiration
	timestampInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}

	tokenTime := time.Unix(timestampInt, 0)
	return time.Since(tokenTime) <= s.cfg.SessionTTL
}

// sessionSecret returns the secret key for session signing,
// derived from AuthHash to avoid requiring extra config
func (s Server) sessionSecret() []byte {
	h := sha256.Sum256([]byte(s.cfg.AuthHash))
	return h[:]
}
