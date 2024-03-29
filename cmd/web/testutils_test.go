package main

import (
	"bytes"
	"html"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/NPeykov/snippetbox/internal/models/mocks"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
)

type testServer struct {
    *httptest.Server
}

var csrfTokenRX = regexp.MustCompile(`<input type="hidden" name="csrf_token" value="(.+)">`)

func newTestApplication(t *testing.T) *application {
    templateCache, err := newTemplateCache()
    if err != nil {
        t.Fatal(err)
    }

    formDecoder := form.NewDecoder()

    sessionManager := scs.New()
    sessionManager.Lifetime = 12 * time.Hour
    sessionManager.Cookie.Secure = true

    return &application{
        errorLog: log.New(io.Discard, "", 0),
        infoLog: log.New(io.Discard, "", 0),
        snippets: &mocks.SnippetModel{},
        users: &mocks.UserModel{},
        templateCache: templateCache,
        formDecoder: formDecoder,
        sessionManager: sessionManager,
    }
}

func newTestServer(t *testing.T, h http.Handler) *testServer {
    ts := httptest.NewTLSServer(h)
    jar, err := cookiejar.New(nil)
    if err != nil {
        t.Fatal()
    }

    ts.Client().Jar = jar
    ts.Client().CheckRedirect = func(req *http.Request, via []*http.Request) error {
        return http.ErrUseLastResponse
    }

    return &testServer{ts}
}

//uses GET request to urlPath, returning status code, haders and body
func (ts *testServer) get(t *testing.T, urlPath string) (int, http.Header, string) {
    rs, err := ts.Client().Get(ts.URL + urlPath)
    if err != nil {
        t.Fatal(err)
    }
    defer rs.Body.Close()
    body, err := io.ReadAll(rs.Body)
    if err != nil {
        t.Fatal(err)
    }
    bytes.TrimSpace(body)
    return rs.StatusCode, rs.Header, string(body)
}

//sends POST request using 'form' as form data to send in the request body
func (ts *testServer) postForm(t *testing.T, urlPath string, form url.Values) (int, http.Header, string) {
    rs, err := ts.Client().PostForm(ts.URL + urlPath, form)
    if err != nil {
        t.Fatal(err)
    }
    defer rs.Body.Close()
    body, err := io.ReadAll(rs.Body)
    if err != nil {
        t.Fatal(err)
    }
    bytes.TrimSpace(body)
    return rs.StatusCode, rs.Header, string(body)
}

func extractCSRFToken(t *testing.T, body string) string {
    matches := csrfTokenRX.FindStringSubmatch(body)
    if len(matches) < 2 {
        t.Fatal("no csrf token found")
    }

    return html.UnescapeString(string(matches[1]))
}

