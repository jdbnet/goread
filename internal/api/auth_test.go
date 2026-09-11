package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	appdb "ebook-reader/internal/db"
)

func testAPI(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	d, err := appdb.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	ts := httptest.NewServer((&Server{DB: d, DataDir: t.TempDir()}).Handler())
	t.Cleanup(ts.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return ts, &http.Client{Jar: jar}
}

func decodeJSON(t *testing.T, res *http.Response, dest any) {
	t.Helper()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		t.Fatalf("decode %s: %v\n%s", res.Status, err, body)
	}
}

func postJSON(t *testing.T, client *http.Client, url, method string, payload any) *http.Response {
	t.Helper()
	var rdr io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		t.Fatal(err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestAuthOptionalThenRequired(t *testing.T) {
	ts, client := testAPI(t)

	res, err := client.Get(ts.URL + "/api/v1/library/books")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("open library: %s", res.Status)
	}
	res.Body.Close()

	res = postJSON(t, client, ts.URL+"/api/v1/auth/credentials", http.MethodPut, map[string]string{
		"username": "jamie",
		"password": "secret12",
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("enable login: %s", res.Status)
	}
	var st authStatusJSON
	decodeJSON(t, res, &st)
	if !st.Enabled || !st.Authenticated || st.Username != "jamie" {
		t.Fatalf("enable login status: %+v", st)
	}

	bare := &http.Client{}
	res, err = bare.Get(ts.URL + "/api/v1/library/books")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("library without session: %s", res.Status)
	}

	res, err = client.Get(ts.URL + "/api/v1/library/books")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("library with session: %s", res.Status)
	}

	res, err = client.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("healthz: %s", res.Status)
	}
}

func TestLoginAndLogout(t *testing.T) {
	ts, owner := testAPI(t)
	res := postJSON(t, owner, ts.URL+"/api/v1/auth/credentials", http.MethodPut, map[string]string{
		"username": "jamie",
		"password": "secret12",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("enable: %s", res.Status)
	}

	guestJar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	guest := &http.Client{Jar: guestJar}

	res = postJSON(t, guest, ts.URL+"/api/v1/auth/login", http.MethodPost, map[string]string{
		"username": "jamie",
		"password": "wrong-password",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad password: %s", res.Status)
	}

	res = postJSON(t, guest, ts.URL+"/api/v1/auth/login", http.MethodPost, map[string]string{
		"username": "jamie",
		"password": "secret12",
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login: %s", res.Status)
	}
	res.Body.Close()

	res, err = guest.Get(ts.URL + "/api/v1/stats")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("stats after login: %s", res.Status)
	}

	res = postJSON(t, guest, ts.URL+"/api/v1/auth/logout", http.MethodPost, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("logout: %s", res.Status)
	}

	res, err = guest.Get(ts.URL + "/api/v1/stats")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("stats after logout: %s", res.Status)
	}
}

func TestChangeAndDisableCredentials(t *testing.T) {
	ts, client := testAPI(t)
	res := postJSON(t, client, ts.URL+"/api/v1/auth/credentials", http.MethodPut, map[string]string{
		"username": "jamie",
		"password": "secret12",
	})
	res.Body.Close()

	res = postJSON(t, client, ts.URL+"/api/v1/auth/credentials", http.MethodPut, map[string]string{
		"username":         "other",
		"password":         "",
		"current_password": "secret12",
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("rename: %s", res.Status)
	}
	var st authStatusJSON
	decodeJSON(t, res, &st)
	if st.Username != "other" {
		t.Fatalf("username after rename: %+v", st)
	}

	res = postJSON(t, client, ts.URL+"/api/v1/auth/disable", http.MethodPost, map[string]string{
		"password": "wrong",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("disable with wrong password: %s", res.Status)
	}

	res = postJSON(t, client, ts.URL+"/api/v1/auth/disable", http.MethodPost, map[string]string{
		"password": "secret12",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("disable: %s", res.Status)
	}

	bare := &http.Client{}
	res, err := bare.Get(ts.URL + "/api/v1/library/books")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("library after disable: %s", res.Status)
	}
}

func TestRejectShortPassword(t *testing.T) {
	ts, client := testAPI(t)
	res := postJSON(t, client, ts.URL+"/api/v1/auth/credentials", http.MethodPut, map[string]string{
		"username": "jamie",
		"password": "short",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("short password: %s", res.Status)
	}
}

func TestAuthStatusAccentWhenLoggedOut(t *testing.T) {
	ts, client := testAPI(t)
	res := postJSON(t, client, ts.URL+"/api/v1/settings", http.MethodPut, map[string]any{
		"font_size":   18,
		"line_height": 1.6,
		"theme":       "light",
		"accent":      "teal",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("save settings: %s", res.Status)
	}

	res = postJSON(t, client, ts.URL+"/api/v1/auth/credentials", http.MethodPut, map[string]string{
		"username": "jamie",
		"password": "secret12",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("enable: %s", res.Status)
	}

	bare := &http.Client{}
	res, err := bare.Get(ts.URL + "/api/v1/auth/status")
	if err != nil {
		t.Fatal(err)
	}
	var st authStatusJSON
	decodeJSON(t, res, &st)
	if st.Authenticated {
		t.Fatal("logged-out status should not be authenticated")
	}
	if st.Accent != "teal" {
		t.Fatalf("logged-out accent %q", st.Accent)
	}
}
