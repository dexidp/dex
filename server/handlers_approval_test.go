package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
)

type getAuthRequestErrorStorage struct {
	storage.Storage
	err error
}

func (s *getAuthRequestErrorStorage) GetAuthRequest(context.Context, string) (storage.AuthRequest, error) {
	return storage.AuthRequest{}, s.err
}

func TestHandleApprovalGetAuthRequestErrorGET(t *testing.T) {
	httpServer, server := newTestServer(t, func(c *Config) {
		c.Storage = &getAuthRequestErrorStorage{Storage: c.Storage, err: errors.New("storage unavailable")}
	})
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/approval?req=any&hmac=AQ", nil)

	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Contains(t, rr.Body.String(), "Database error.")
}

func TestHandleApprovalGetAuthRequestNotFoundGET(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/approval?req=does-not-exist&hmac=AQ", nil)

	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "User session error.")
	require.NotContains(t, rr.Body.String(), "Database error.")
}

func TestHandleApprovalGetAuthRequestNotFoundPOST(t *testing.T) {
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	body := strings.NewReader("approval=approve&req=does-not-exist&hmac=AQ")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/approval", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	server.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "User session error.")
	require.NotContains(t, rr.Body.String(), "Database error.")
}

func TestHandleApprovalDoubleSubmitPOST(t *testing.T) {
	ctx := t.Context()
	httpServer, server := newTestServer(t, nil)
	defer httpServer.Close()

	authReq := storage.AuthRequest{
		ID:            "approval-double-submit",
		ClientID:      "test",
		ResponseTypes: []string{responseTypeCode},
		RedirectURI:   "https://client.example/callback",
		Expiry:        time.Now().Add(time.Minute),
		LoggedIn:      true,
		MFAValidated:  true,
		HMACKey:       []byte("approval-double-submit-key"),
	}
	require.NoError(t, server.storage.CreateAuthRequest(ctx, authReq))

	h := hmac.New(sha256.New, authReq.HMACKey)
	h.Write([]byte(authReq.ID))
	mac := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	form := url.Values{
		"approval": {"approve"},
		"req":      {authReq.ID},
		"hmac":     {mac},
	}

	firstRR := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodPost, "/approval", strings.NewReader(form.Encode()))
	firstReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(firstRR, firstReq)

	require.Equal(t, http.StatusSeeOther, firstRR.Code)
	require.Contains(t, firstRR.Header().Get("Location"), "https://client.example/callback")

	secondRR := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodPost, "/approval", strings.NewReader(form.Encode()))
	secondReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(secondRR, secondReq)

	require.Equal(t, http.StatusBadRequest, secondRR.Code)
	require.Contains(t, secondRR.Body.String(), "User session error.")
	require.NotContains(t, secondRR.Body.String(), "Database error.")
}
