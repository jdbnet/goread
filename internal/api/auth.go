package api

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	appdb "ebook-reader/internal/db"
)

const (
	sessionCookie = "ebook_session"
	sessionTTL    = 30 * 24 * time.Hour
)

type authStatusJSON struct {
	Enabled       bool   `json:"enabled"`
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username"`
	Accent        string `json:"accent"`
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if publicPath(r) {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		enabled, err := s.DB.AuthEnabled()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if !enabled {
			next.ServeHTTP(w, r)
			return
		}
		ok, err := s.DB.ValidSession(sessionToken(r))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func publicPath(r *http.Request) bool {
	switch r.URL.Path {
	case "/healthz":
		return r.Method == http.MethodGet || r.Method == http.MethodHead
	case "/api/v1/auth/status":
		return r.Method == http.MethodGet
	case "/api/v1/auth/login":
		return r.Method == http.MethodPost
	case "/api/v1/auth/logout":
		return r.Method == http.MethodPost
	default:
		return false
	}
}

func (s *Server) getAuthStatus(w http.ResponseWriter, r *http.Request) {
	enabled, err := s.DB.AuthEnabled()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	st := authStatusJSON{Enabled: enabled}
	if !enabled {
		s.writeAuthStatus(w, st)
		return
	}
	ok, err := s.DB.ValidSession(sessionToken(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	st.Authenticated = ok
	if ok {
		c, err := s.DB.GetAuth()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		st.Username = c.Username
	}
	s.writeAuthStatus(w, st)
}

func (s *Server) postLogin(w http.ResponseWriter, r *http.Request) {
	enabled, err := s.DB.AuthEnabled()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !enabled {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("login is not enabled"))
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}
	c, err := s.DB.GetAuth()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	userOK := subtleEqualString(strings.TrimSpace(body.Username), c.Username)
	passOK := appdb.VerifyPassword(c.PasswordHash, body.Password)
	if !userOK || !passOK {
		writeErr(w, http.StatusUnauthorized, fmt.Errorf("invalid username or password"))
		return
	}
	if err := s.issueSession(w, r); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.writeAuthStatus(w, authStatusJSON{
		Enabled:       true,
		Authenticated: true,
		Username:      c.Username,
	})
}

func (s *Server) postLogout(w http.ResponseWriter, r *http.Request) {
	if token := sessionToken(r); token != "" {
		_ = s.DB.DeleteSession(token)
	}
	clearSessionCookie(w, r)
	enabled, err := s.DB.AuthEnabled()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.writeAuthStatus(w, authStatusJSON{Enabled: enabled})
}

func (s *Server) putCredentials(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username        string `json:"username"`
		Password        string `json:"password"`
		CurrentPassword string `json:"current_password"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}
	enabled, err := s.DB.AuthEnabled()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	c, err := s.DB.GetAuth()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if enabled && !appdb.VerifyPassword(c.PasswordHash, body.CurrentPassword) {
		writeErr(w, http.StatusForbidden, fmt.Errorf("current password is incorrect"))
		return
	}
	username := strings.TrimSpace(body.Username)
	if err := appdb.ValidateUsername(username); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if enabled && body.Password == "" {
		if err := s.DB.UpdateUsername(username); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		s.writeAuthStatus(w, authStatusJSON{
			Enabled:       true,
			Authenticated: true,
			Username:      username,
		})
		return
	}
	if err := s.DB.SetCredentials(username, body.Password); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := s.DB.DeleteAllSessions(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.issueSession(w, r); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	s.writeAuthStatus(w, authStatusJSON{
		Enabled:       true,
		Authenticated: true,
		Username:      username,
	})
}

func (s *Server) postDisableAuth(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}
	enabled, err := s.DB.AuthEnabled()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !enabled {
		s.writeAuthStatus(w, authStatusJSON{})
		return
	}
	c, err := s.DB.GetAuth()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !appdb.VerifyPassword(c.PasswordHash, body.Password) {
		writeErr(w, http.StatusForbidden, fmt.Errorf("current password is incorrect"))
		return
	}
	if err := s.DB.ClearCredentials(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.DB.DeleteAllSessions(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	clearSessionCookie(w, r)
	s.writeAuthStatus(w, authStatusJSON{})
}

func (s *Server) writeAuthStatus(w http.ResponseWriter, st authStatusJSON) {
	settings, err := s.DB.GetSettings()
	if err != nil {
		st.Accent = "amber"
	} else {
		st.Accent = settings.Accent
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request) error {
	token, exp, err := s.DB.CreateSession(sessionTTL)
	if err != nil {
		return err
	}
	setSessionCookie(w, r, token, exp)
	return nil
}

func sessionToken(r *http.Request) string {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  exp,
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS(r),
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS(r),
	})
}

func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func subtleEqualString(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
