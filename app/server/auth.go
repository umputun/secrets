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

	log "github.com/go-pkgz/lgr"
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

	if !usernameCorrect || !passwordCorrect {
		log.Printf("[WARN] basic auth failed, ip=%s", GetHashedIP(r))
		return false
	}
	return true
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
		log.Printf("[WARN] login failed, ip=%s", GetHashedIP(r))
		s.renderLoginPopup(w, r, "invalid password")
		return
	}

	log.Printf("[INFO] login success, ip=%s", GetHashedIP(r))
	// authentication successful, set session cookie
	//nolint:gosec // G124: Secure is set from the configured protocol, plain http is allowed for local runs
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
	//nolint:gosec // G124: Secure is set from the configured protocol, plain http is allowed for local runs
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

	secret := s.sessionSecret()

	h := hmac.New(sha256.New, secret)
	h.Write([]byte(tokenID + "." + timestamp))
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
	if _, err := uuid.Parse(tokenID); err != nil {
		return false
	}

	// recreate signature
	secret := s.sessionSecret()
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(tokenID + "." + timestamp))
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
	age := time.Since(tokenTime)
	return age >= -30*time.Second && age <= s.cfg.SessionTTL
}

func (s Server) sessionSecret() []byte {
	h := hmac.New(sha256.New, []byte(s.cfg.SignKey))
	h.Write([]byte("secrets:session:" + s.cfg.AuthHash))
	return h.Sum(nil)
}
