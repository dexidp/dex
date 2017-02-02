package main

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

func TestCookieStore(t *testing.T) {
	store := &cookieStore{}

	val := []byte("foobar")

	hf := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/set":
			if err := store.setCookie(w, "name", val); err != nil {
				t.Errorf("set cookie: %v", err)
			}
		default:
			got := store.cookie(r, "name")
			if got == nil {
				t.Error("didn't get a cookie")
			} else {
				if !bytes.Equal(got, val) {
					t.Errorf("decoded cookie didn't match encoded value")
				}
			}
		}
		w.Write([]byte("ok"))
	}

	s := httptest.NewTLSServer(http.HandlerFunc(hf))
	defer s.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}

	c := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Jar: jar,
	}

	if _, err := c.Get(s.URL + "/set"); err != nil {
		t.Error(err)
	}
	if _, err := c.Get(s.URL + "/get"); err != nil {
		t.Error(err)
	}
}
