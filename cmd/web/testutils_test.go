package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

type testServer struct {
    *httptest.Server
}

func newTestApplication() *application {
    return &application{
        errorLog: log.New(io.Discard, "", 0),
        infoLog: log.New(io.Discard, "", 0),
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

