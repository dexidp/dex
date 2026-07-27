package device

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/dexidp/dex/server/connectors"
	"github.com/dexidp/dex/server/grants"
	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/server/router"
	"github.com/dexidp/dex/server/templates"
	"github.com/dexidp/dex/server/tokens"
	"github.com/dexidp/dex/storage"
)

// DeviceCodeResponse is the device authorization response (RFC 8628 §3.2).
type DeviceCodeResponse struct {
	// The unique device code for device authentication
	DeviceCode string `json:"device_code"`
	// The code the user will exchange via a browser and log in
	UserCode string `json:"user_code"`
	// The url to verify the user code.
	VerificationURI string `json:"verification_uri"`
	// The verification uri with the user code appended for pre-filling form
	VerificationURIComplete string `json:"verification_uri_complete"`
	// The lifetime of the device code
	ExpireTime int `json:"expires_in"`
	// How often the device is allowed to poll to verify that the user login occurred
	PollInterval int `json:"interval"`
}

// Handler serves the browser side of the device authorization grant.
type Handler struct {
	IssuerURL        oauth2.IssuerURL
	Storage          storage.Storage
	Templates        *templates.Templates
	Now              func() time.Time
	RequestsValidFor time.Duration
	Logger           *slog.Logger

	// Issuer mints the tokens, and Connectors resolves the connector, for the
	// auth-code exchange the device flow shares with the authorization_code grant
	// via grants.ExchangeAuthCode.
	Issuer     *tokens.Issuer
	Connectors *connectors.Cache
}

// Mount registers the device authorization routes.
func (h *Handler) Mount(m router.Mux) {
	m.HandleFunc("/device", h.handleDeviceExchange)
	m.HandleFunc("/device/auth/verify_code", h.verifyUserCode)
	m.HandleFunc("/device/code", h.handleDeviceCode)
	m.HandleFunc(oauth2.DeviceCallbackURI, h.handleDeviceCallback)
}

// deviceFlowError is a failed step in the flow. A non-empty OAuth2 code makes the
// handler write a JSON error response; otherwise the message is rendered as an
// HTML error page.
type deviceFlowError struct {
	status  int
	code    string
	message string
}

func (h *Handler) writeFlowError(r *http.Request, w http.ResponseWriter, e *deviceFlowError) {
	if e.code != "" {
		h.writeError(w, e.code, e.message, e.status)
		return
	}
	h.renderError(r, w, e.status, e.message)
}

// writeError writes a JSON OAuth2 error response.
func (h *Handler) writeError(w http.ResponseWriter, typ, description string, statusCode int) {
	if err := oauth2.WriteError(w, typ, description, statusCode); err != nil {
		h.Logger.Error("device error response", "err", err)
	}
}

// renderError renders an HTML error page.
func (h *Handler) renderError(r *http.Request, w http.ResponseWriter, status int, description string) {
	if err := h.Templates.Err(r, w, status, description); err != nil {
		h.Logger.ErrorContext(r.Context(), "server template error", "err", err)
	}
}

func (h *Handler) getDeviceVerificationURI() string {
	return h.IssuerURL.AbsPath("/device/auth/verify_code")
}

// handleDeviceExchange serves the /device user-code entry page.
func (h *Handler) handleDeviceExchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
		return
	}

	// If "user_code" is set, pre-populate the user code field. If "invalid" is
	// set, show a message that the code was invalid or expired.
	userCode := r.URL.Query().Get("user_code")
	invalidAttempt, err := strconv.ParseBool(r.URL.Query().Get("invalid"))
	if err != nil {
		invalidAttempt = false
	}
	if err := h.Templates.Device(r, w, h.getDeviceVerificationURI(), userCode, invalidAttempt); err != nil {
		h.Logger.ErrorContext(r.Context(), "server template error", "err", err)
		h.renderError(r, w, http.StatusNotFound, "Page not found")
	}
}

// deviceCodeRequest is a parsed /device/code authorization request.
type deviceCodeRequest struct {
	clientID     string
	clientSecret string
	scopes       []string
	pkce         storage.PKCE
}

// parseDeviceCodeRequest parses and validates the /device/code form.
func (h *Handler) parseDeviceCodeRequest(r *http.Request) (deviceCodeRequest, *deviceFlowError) {
	if err := r.ParseForm(); err != nil {
		h.Logger.ErrorContext(r.Context(), "could not parse Device Request body", "err", err)
		return deviceCodeRequest{}, &deviceFlowError{status: http.StatusNotFound, code: oauth2.InvalidRequest}
	}

	method := r.Form.Get("code_challenge_method")
	if method == "" {
		method = oauth2.PKCEMethodPlain
	}
	if method != oauth2.PKCEMethodS256 && method != oauth2.PKCEMethodPlain {
		return deviceCodeRequest{}, &deviceFlowError{
			status:  http.StatusBadRequest,
			code:    oauth2.InvalidRequest,
			message: fmt.Sprintf("Unsupported PKCE challenge method (%q).", method),
		}
	}

	scopes := strings.Fields(r.Form.Get("scope"))
	if len(scopes) == 0 {
		// per RFC 8628 §3.1 scope is optional, but dex requires at least 'openid'.
		scopes = []string{"openid"}
	}

	return deviceCodeRequest{
		clientID:     r.Form.Get("client_id"),
		clientSecret: r.Form.Get("client_secret"),
		scopes:       scopes,
		pkce: storage.PKCE{
			CodeChallenge:       r.Form.Get("code_challenge"),
			CodeChallengeMethod: method,
		},
	}, nil
}

// createDeviceAuthorization mints and stores the device and user codes and builds
// the authorization response the device polls against.
func (h *Handler) createDeviceAuthorization(ctx context.Context, req deviceCodeRequest) (*DeviceCodeResponse, *deviceFlowError) {
	h.Logger.InfoContext(ctx, "received device request", "client_id", req.clientID, "scoped", req.scopes)

	deviceCode := storage.NewDeviceCode()
	userCode := storage.NewUserCode()
	expireTime := time.Now().Add(h.RequestsValidFor)

	if err := h.Storage.CreateDeviceRequest(ctx, storage.DeviceRequest{
		UserCode:     userCode,
		DeviceCode:   deviceCode,
		ClientID:     req.clientID,
		ClientSecret: req.clientSecret,
		Scopes:       req.scopes,
		Expiry:       expireTime,
	}); err != nil {
		h.Logger.ErrorContext(ctx, "failed to store device request", "err", err)
		return nil, &deviceFlowError{status: http.StatusInternalServerError, code: oauth2.InvalidRequest}
	}

	if err := h.Storage.CreateDeviceToken(ctx, storage.DeviceToken{
		DeviceCode:          deviceCode,
		Status:              oauth2.DeviceTokenPending,
		Expiry:              expireTime,
		LastRequestTime:     h.Now(),
		PollIntervalSeconds: 0,
		PKCE:                req.pkce,
	}); err != nil {
		h.Logger.ErrorContext(ctx, "failed to store device token", "err", err)
		return nil, &deviceFlowError{status: http.StatusInternalServerError, code: oauth2.InvalidRequest}
	}

	u := h.IssuerURL
	u.Path = path.Join(u.Path, "device")
	vURI := u.String()

	q := u.Query()
	q.Set("user_code", userCode)
	u.RawQuery = q.Encode()
	vURIComplete := u.String()

	return &DeviceCodeResponse{
		DeviceCode:              deviceCode,
		UserCode:                userCode,
		VerificationURI:         vURI,
		VerificationURIComplete: vURIComplete,
		ExpireTime:              int(h.RequestsValidFor.Seconds()),
		PollInterval:            5,
	}, nil
}

func (h *Handler) handleDeviceCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.renderError(r, w, http.StatusBadRequest, "Invalid device code request type")
		h.writeError(w, oauth2.InvalidRequest, "", http.StatusBadRequest)
		return
	}

	req, ferr := h.parseDeviceCodeRequest(r)
	if ferr != nil {
		h.writeFlowError(r, w, ferr)
		return
	}

	resp, ferr := h.createDeviceAuthorization(r.Context(), req)
	if ferr != nil {
		h.writeFlowError(r, w, ferr)
		return
	}

	writeDeviceCodeResponse(w, resp)
}

// writeDeviceCodeResponse writes the device authorization response: it can carry
// a cache-control header (RFC 8628 §3.2) and is JSON (RFC 6749 §5.1).
func writeDeviceCodeResponse(w http.ResponseWriter, resp *DeviceCodeResponse) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) verifyUserCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.renderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
		return
	}
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		h.Logger.Warn("could not parse user code verification request body", "err", err)
		h.renderError(r, w, http.StatusBadRequest, "")
		return
	}

	userCode := r.Form.Get("user_code")
	if userCode == "" {
		h.renderError(r, w, http.StatusBadRequest, "No user code received")
		return
	}
	userCode = strings.ToUpper(userCode)

	// Find the user code among the outstanding requests.
	deviceRequest, err := h.Storage.GetDeviceRequest(ctx, userCode)
	if err != nil || h.Now().After(deviceRequest.Expiry) {
		if err != nil && err != storage.ErrNotFound {
			h.Logger.ErrorContext(ctx, "failed to get device request", "err", err)
		}
		if err := h.Templates.Device(r, w, h.getDeviceVerificationURI(), userCode, true); err != nil {
			h.Logger.ErrorContext(ctx, "Server template error", "err", err)
			h.renderError(r, w, http.StatusNotFound, "Page not found")
		}
		return
	}

	// Redirect to the dex auth endpoint, which sends the user back to the device
	// callback once they authenticate.
	u := h.IssuerURL
	u.Path = path.Join(u.Path, "/auth")
	q := u.Query()
	q.Set("client_id", deviceRequest.ClientID)
	// Do not put client_secret in this browser redirect: /auth is the
	// authorization endpoint and never consumes it, so it would only leak the
	// confidential secret into browser history, Referer, and access logs. The
	// client is authenticated later in completeDeviceAuthorization against the
	// stored device request.
	q.Set("state", deviceRequest.UserCode)
	q.Set("response_type", "code")
	q.Set("redirect_uri", h.IssuerURL.AbsPath(oauth2.DeviceCallbackURI))
	q.Set("scope", strings.Join(deviceRequest.Scopes, " "))
	u.RawQuery = q.Encode()

	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (h *Handler) handleDeviceCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.Logger.ErrorContext(r.Context(), "unsupported method in device callback", "method", r.Method)
		h.renderError(r, w, http.StatusBadRequest, "Method not allowed.")
		return
	}

	clientName, ferr := h.completeDeviceAuthorization(w, r)
	if ferr != nil {
		h.writeFlowError(r, w, ferr)
		return
	}

	if err := h.Templates.DeviceSuccess(r, w, clientName); err != nil {
		h.Logger.ErrorContext(r.Context(), "Server template error", "err", err)
		h.renderError(r, w, http.StatusNotFound, "Page not found")
	}
}

// completeDeviceAuthorization handles the browser callback: it exchanges the
// authorization code for tokens and stores them against the device code so the
// polling device_code grant can return them. It returns the client name for the
// success page.
func (h *Handler) completeDeviceAuthorization(w http.ResponseWriter, r *http.Request) (string, *deviceFlowError) {
	ctx := r.Context()

	userCode := r.FormValue("state")
	code := r.FormValue("code")
	if userCode == "" || code == "" {
		return "", &deviceFlowError{status: http.StatusBadRequest, message: "Request was missing parameters"}
	}

	// Authorization redirect callback from the OAuth2 auth flow.
	if errMsg := r.FormValue("error"); errMsg != "" {
		// Log the error details but don't expose them to the user.
		h.Logger.ErrorContext(ctx, "OAuth2 authorization error",
			"error", errMsg,
			"error_description", r.FormValue("error_description"))
		return "", &deviceFlowError{status: http.StatusBadRequest, message: "Authorization failed. Please try again."}
	}

	authCode, err := h.Storage.GetAuthCode(ctx, code)
	if err != nil || h.Now().After(authCode.Expiry) {
		status := http.StatusBadRequest
		if err != nil && err != storage.ErrNotFound {
			h.Logger.ErrorContext(ctx, "failed to get auth code", "err", err)
			status = http.StatusInternalServerError
		}
		return "", &deviceFlowError{status: status, message: "Invalid or expired auth code."}
	}

	deviceReq, err := h.Storage.GetDeviceRequest(ctx, userCode)
	if err != nil || h.Now().After(deviceReq.Expiry) {
		status := http.StatusBadRequest
		if err != nil && err != storage.ErrNotFound {
			h.Logger.ErrorContext(ctx, "failed to get device code", "err", err)
			status = http.StatusInternalServerError
		}
		return "", &deviceFlowError{status: status, message: "Invalid or expired user code."}
	}

	// Bind the auth code to this device request: it must have been minted for the
	// same client and issued to the device callback redirect. The authorization_code
	// grant enforces the same client/redirect binding (see grants/authcode.go); the
	// device callback must not skip it, or a code minted for one client could be
	// redeemed against another client's device request (cross-client token theft).
	// The redirect is matched on its parsed path suffix, mirroring how the auth flow
	// recognizes the device callback: the issuer path prefix does not matter, and a
	// "/device/callback" in the query string can not spoof it. A value that fails to
	// parse is not a valid device redirect.
	redirectURL, err := url.Parse(authCode.RedirectURI)
	validRedirect := err == nil && strings.HasSuffix(redirectURL.Path, oauth2.DeviceCallbackURI)
	if authCode.ClientID != deviceReq.ClientID || !validRedirect {
		h.Logger.ErrorContext(ctx, "device callback: auth code does not match the device request",
			"auth_code_client_id", authCode.ClientID, "device_client_id", deviceReq.ClientID)
		return "", &deviceFlowError{status: http.StatusBadRequest, message: "Invalid or expired auth code."}
	}

	client, err := h.Storage.GetClient(ctx, deviceReq.ClientID)
	if err != nil {
		if err != storage.ErrNotFound {
			h.Logger.ErrorContext(ctx, "failed to get client", "err", err)
			return "", &deviceFlowError{status: http.StatusInternalServerError, code: oauth2.ServerError}
		}
		return "", &deviceFlowError{status: http.StatusUnauthorized, code: oauth2.InvalidClient, message: "Invalid client credentials."}
	}
	// Constant-time comparison of the client secret, matching grants.go's client
	// authentication, so the compare does not leak the secret via timing.
	if subtle.ConstantTimeCompare([]byte(client.Secret), []byte(deviceReq.ClientSecret)) != 1 {
		return "", &deviceFlowError{status: http.StatusUnauthorized, code: oauth2.InvalidClient, message: "Invalid client credentials."}
	}

	// ExchangeAuthCode consumes the code (its atomic single-use gate) and returns
	// what to issue; the tokens are minted here.
	auth, withRefresh, err := grants.ExchangeAuthCode(ctx, h.Storage, h.Connectors, h.Logger, authCode, client)
	if err != nil {
		h.Logger.ErrorContext(ctx, "could not exchange auth code for client", "client_id", deviceReq.ClientID, "err", err)
		return "", &deviceFlowError{status: http.StatusInternalServerError, message: "Failed to exchange auth code."}
	}
	resp, err := h.Issuer.IssueResponse(ctx, auth, authCode.ID, withRefresh)
	if err != nil {
		h.Logger.ErrorContext(ctx, "could not issue tokens for device flow", "client_id", deviceReq.ClientID, "err", err)
		return "", &deviceFlowError{status: http.StatusInternalServerError, message: "Failed to exchange auth code."}
	}

	old, err := h.Storage.GetDeviceToken(ctx, deviceReq.DeviceCode)
	if err != nil || h.Now().After(old.Expiry) {
		status := http.StatusBadRequest
		if err != nil && err != storage.ErrNotFound {
			h.Logger.ErrorContext(ctx, "failed to get device token", "err", err)
			status = http.StatusInternalServerError
		}
		return "", &deviceFlowError{status: status, message: "Invalid or expired device code."}
	}

	// Store the token against the device code and mark it complete.
	updater := func(old storage.DeviceToken) (storage.DeviceToken, error) {
		if old.Status == oauth2.DeviceTokenComplete {
			return old, errors.New("device token already complete")
		}
		respStr, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			h.Logger.ErrorContext(ctx, "failed to marshal device token response", "err", err)
			h.renderError(r, w, http.StatusInternalServerError, "")
			return old, err
		}
		old.Token = string(respStr)
		old.Status = oauth2.DeviceTokenComplete
		return old, nil
	}
	if err := h.Storage.UpdateDeviceToken(ctx, deviceReq.DeviceCode, updater); err != nil {
		h.Logger.ErrorContext(ctx, "failed to update device token", "err", err)
		return "", &deviceFlowError{status: http.StatusBadRequest, message: ""}
	}

	return client.Name, nil
}
