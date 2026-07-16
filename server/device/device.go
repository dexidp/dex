// Package device implements the OAuth2 device authorization grant (RFC 8628):
// the /device user-code entry page, the /device/code authorization request,
// user-code verification, the browser callback that completes the flow, and the
// device_code token grant that the device polls.
package device

import (
	"context"
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

// Handler serves the device authorization grant. ExchangeAuthCode and
// CalculateCodeChallenge are shared with the authorization-code grant;
// RenderError renders an HTML error page. They are supplied by the server so the
// handler does not depend on the whole Server.
type Handler struct {
	IssuerURL              url.URL
	AbsURL                 func(...string) string
	AbsPath                func(...string) string
	Storage                storage.Storage
	Templates              *templates.Templates
	Now                    func() time.Time
	RequestsValidFor       time.Duration
	Logger                 *slog.Logger
	RenderError            func(*http.Request, http.ResponseWriter, int, string)
	ExchangeAuthCode       func(ctx context.Context, w http.ResponseWriter, authCode storage.AuthCode, client storage.Client) (tokens.Response, error)
	CalculateCodeChallenge func(codeVerifier, codeChallengeMethod string) (string, error)
}

// Mount registers the device-flow routes.
func (h *Handler) Mount(m router.Mux) {
	m.HandleFunc("/device", h.handleDeviceExchange)
	m.HandleFunc("/device/auth/verify_code", h.verifyUserCode)
	m.HandleFunc("/device/code", h.handleDeviceCode)
	m.HandleFunc("/device/token", h.handleDeviceTokenDeprecated)
	m.HandleFunc(oauth2.DeviceCallbackURI, h.handleDeviceCallback)
}

func (h *Handler) writeError(w http.ResponseWriter, typ, description string, statusCode int) {
	if err := oauth2.WriteError(w, typ, description, statusCode); err != nil {
		h.Logger.Error("device error response", "err", err)
	}
}

func (h *Handler) getDeviceVerificationURI() string {
	return path.Join(h.IssuerURL.Path, "/device/auth/verify_code")
}

func (h *Handler) handleDeviceExchange(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Grab the parameter(s) from the query.
		// If "user_code" is set, pre-populate the user code text field.
		// If "invalid" is set, set the invalidAttempt boolean, which will display a message to the user that they
		// attempted to redeem an invalid or expired user code.
		userCode := r.URL.Query().Get("user_code")
		invalidAttempt, err := strconv.ParseBool(r.URL.Query().Get("invalid"))
		if err != nil {
			invalidAttempt = false
		}
		if err := h.Templates.Device(r, w, h.getDeviceVerificationURI(), userCode, invalidAttempt); err != nil {
			h.Logger.ErrorContext(r.Context(), "server template error", "err", err)
			h.RenderError(r, w, http.StatusNotFound, "Page not found")
		}
	default:
		h.RenderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
	}
}

func (h *Handler) handleDeviceCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pollIntervalSeconds := 5

	switch r.Method {
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			h.Logger.ErrorContext(r.Context(), "could not parse Device Request body", "err", err)
			h.writeError(w, oauth2.InvalidRequest, "", http.StatusNotFound)
			return
		}

		// Get the client id and scopes from the post
		clientID := r.Form.Get("client_id")
		clientSecret := r.Form.Get("client_secret")
		scopes := strings.Fields(r.Form.Get("scope"))
		codeChallenge := r.Form.Get("code_challenge")
		codeChallengeMethod := r.Form.Get("code_challenge_method")

		if codeChallengeMethod == "" {
			codeChallengeMethod = oauth2.PKCEMethodPlain
		}
		if codeChallengeMethod != oauth2.PKCEMethodS256 && codeChallengeMethod != oauth2.PKCEMethodPlain {
			description := fmt.Sprintf("Unsupported PKCE challenge method (%q).", codeChallengeMethod)
			h.writeError(w, oauth2.InvalidRequest, description, http.StatusBadRequest)
			return
		}

		if len(scopes) == 0 {
			// per RFC8628 section 3.1, https://datatracker.ietf.org/doc/html/rfc8628#section-3.1
			// scope is optional but dex requires that it is always at least 'openid' so default it
			scopes = []string{"openid"}
		}

		h.Logger.InfoContext(r.Context(), "received device request", "client_id", clientID, "scoped", scopes)

		// Make device code
		deviceCode := storage.NewDeviceCode()

		// make user code
		userCode := storage.NewUserCode()

		// Generate the expire time
		expireTime := time.Now().Add(h.RequestsValidFor)

		// Store the Device Request
		deviceReq := storage.DeviceRequest{
			UserCode:     userCode,
			DeviceCode:   deviceCode,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       scopes,
			Expiry:       expireTime,
		}

		if err := h.Storage.CreateDeviceRequest(ctx, deviceReq); err != nil {
			h.Logger.ErrorContext(r.Context(), "failed to store device request", "err", err)
			h.writeError(w, oauth2.InvalidRequest, "", http.StatusInternalServerError)
			return
		}

		// Store the device token
		deviceToken := storage.DeviceToken{
			DeviceCode:          deviceCode,
			Status:              oauth2.DeviceTokenPending,
			Expiry:              expireTime,
			LastRequestTime:     h.Now(),
			PollIntervalSeconds: 0,
			PKCE: storage.PKCE{
				CodeChallenge:       codeChallenge,
				CodeChallengeMethod: codeChallengeMethod,
			},
		}

		if err := h.Storage.CreateDeviceToken(ctx, deviceToken); err != nil {
			h.Logger.ErrorContext(r.Context(), "failed to store device token", "err", err)
			h.writeError(w, oauth2.InvalidRequest, "", http.StatusInternalServerError)
			return
		}

		u, err := url.Parse(h.IssuerURL.String())
		if err != nil {
			h.Logger.ErrorContext(r.Context(), "could not parse issuer URL", "err", err)
			h.writeError(w, oauth2.InvalidRequest, "", http.StatusInternalServerError)
			return
		}
		u.Path = path.Join(u.Path, "device")
		vURI := u.String()

		q := u.Query()
		q.Set("user_code", userCode)
		u.RawQuery = q.Encode()
		vURIComplete := u.String()

		code := DeviceCodeResponse{
			DeviceCode:              deviceCode,
			UserCode:                userCode,
			VerificationURI:         vURI,
			VerificationURIComplete: vURIComplete,
			ExpireTime:              int(h.RequestsValidFor.Seconds()),
			PollInterval:            pollIntervalSeconds,
		}

		// Device Authorization Response can contain cache control header according to
		// https://tools.ietf.org/html/rfc8628#section-3.2
		w.Header().Set("Cache-Control", "no-store")

		// Response type should be application/json according to
		// https://datatracker.ietf.org/doc/html/rfc6749#section-5.1
		w.Header().Set("Content-Type", "application/json")

		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "   ")
		enc.Encode(code)

	default:
		h.RenderError(r, w, http.StatusBadRequest, "Invalid device code request type")
		h.writeError(w, oauth2.InvalidRequest, "", http.StatusBadRequest)
	}
}

func (h *Handler) verifyUserCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			h.Logger.Warn("could not parse user code verification request body", "err", err)
			h.RenderError(r, w, http.StatusBadRequest, "")
			return
		}

		userCode := r.Form.Get("user_code")
		if userCode == "" {
			h.RenderError(r, w, http.StatusBadRequest, "No user code received")
			return
		}

		userCode = strings.ToUpper(userCode)

		// Find the user code in the available requests
		deviceRequest, err := h.Storage.GetDeviceRequest(ctx, userCode)
		if err != nil || h.Now().After(deviceRequest.Expiry) {
			if err != nil && err != storage.ErrNotFound {
				h.Logger.ErrorContext(r.Context(), "failed to get device request", "err", err)
			}
			if err := h.Templates.Device(r, w, h.getDeviceVerificationURI(), userCode, true); err != nil {
				h.Logger.ErrorContext(r.Context(), "Server template error", "err", err)
				h.RenderError(r, w, http.StatusNotFound, "Page not found")
			}
			return
		}

		// Redirect to Dex Auth Endpoint
		authURL := h.AbsURL("/auth")
		u, err := url.Parse(authURL)
		if err != nil {
			h.RenderError(r, w, http.StatusInternalServerError, "Invalid auth URI.")
			return
		}
		q := u.Query()
		q.Set("client_id", deviceRequest.ClientID)
		q.Set("client_secret", deviceRequest.ClientSecret)
		q.Set("state", deviceRequest.UserCode)
		q.Set("response_type", "code")
		q.Set("redirect_uri", h.AbsPath(oauth2.DeviceCallbackURI))
		q.Set("scope", strings.Join(deviceRequest.Scopes, " "))
		u.RawQuery = q.Encode()

		http.Redirect(w, r, u.String(), http.StatusFound)

	default:
		h.RenderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
	}
}

func (h *Handler) handleDeviceCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		userCode := r.FormValue("state")
		code := r.FormValue("code")

		if userCode == "" || code == "" {
			h.RenderError(r, w, http.StatusBadRequest, "Request was missing parameters")
			return
		}

		// Authorization redirect callback from OAuth2 auth flow.
		if errMsg := r.FormValue("error"); errMsg != "" {
			// Log the error details but don't expose them to the user
			h.Logger.ErrorContext(r.Context(), "OAuth2 authorization error",
				"error", errMsg,
				"error_description", r.FormValue("error_description"))
			h.RenderError(r, w, http.StatusBadRequest, "Authorization failed. Please try again.")
			return
		}

		authCode, err := h.Storage.GetAuthCode(ctx, code)
		if err != nil || h.Now().After(authCode.Expiry) {
			errCode := http.StatusBadRequest
			if err != nil && err != storage.ErrNotFound {
				h.Logger.ErrorContext(r.Context(), "failed to get auth code", "err", err)
				errCode = http.StatusInternalServerError
			}
			h.RenderError(r, w, errCode, "Invalid or expired auth code.")
			return
		}

		// Grab the device request from storage
		deviceReq, err := h.Storage.GetDeviceRequest(ctx, userCode)
		if err != nil || h.Now().After(deviceReq.Expiry) {
			errCode := http.StatusBadRequest
			if err != nil && err != storage.ErrNotFound {
				h.Logger.ErrorContext(r.Context(), "failed to get device code", "err", err)
				errCode = http.StatusInternalServerError
			}
			h.RenderError(r, w, errCode, "Invalid or expired user code.")
			return
		}

		client, err := h.Storage.GetClient(ctx, deviceReq.ClientID)
		if err != nil {
			if err != storage.ErrNotFound {
				h.Logger.ErrorContext(r.Context(), "failed to get client", "err", err)
				h.writeError(w, oauth2.ServerError, "", http.StatusInternalServerError)
			} else {
				h.writeError(w, oauth2.InvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
			}
			return
		}
		if client.Secret != deviceReq.ClientSecret {
			h.writeError(w, oauth2.InvalidClient, "Invalid client credentials.", http.StatusUnauthorized)
			return
		}

		resp, err := h.ExchangeAuthCode(ctx, w, authCode, client)
		if err != nil {
			h.Logger.ErrorContext(r.Context(), "could not exchange auth code for clien", "client_id", deviceReq.ClientID, "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, "Failed to exchange auth code.")
			return
		}

		// Grab the device token from storage
		old, err := h.Storage.GetDeviceToken(ctx, deviceReq.DeviceCode)
		if err != nil || h.Now().After(old.Expiry) {
			errCode := http.StatusBadRequest
			if err != nil && err != storage.ErrNotFound {
				h.Logger.ErrorContext(r.Context(), "failed to get device token", "err", err)
				errCode = http.StatusInternalServerError
			}
			h.RenderError(r, w, errCode, "Invalid or expired device code.")
			return
		}

		updater := func(old storage.DeviceToken) (storage.DeviceToken, error) {
			if old.Status == oauth2.DeviceTokenComplete {
				return old, errors.New("device token already complete")
			}
			respStr, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				h.Logger.ErrorContext(r.Context(), "failed to marshal device token response", "err", err)
				h.RenderError(r, w, http.StatusInternalServerError, "")
				return old, err
			}

			old.Token = string(respStr)
			old.Status = oauth2.DeviceTokenComplete
			return old, nil
		}

		// Update refresh token in the storage, store the token and mark as complete
		if err := h.Storage.UpdateDeviceToken(ctx, deviceReq.DeviceCode, updater); err != nil {
			h.Logger.ErrorContext(r.Context(), "failed to update device token", "err", err)
			h.RenderError(r, w, http.StatusBadRequest, "")
			return
		}

		if err := h.Templates.DeviceSuccess(r, w, client.Name); err != nil {
			h.Logger.ErrorContext(r.Context(), "Server template error", "err", err)
			h.RenderError(r, w, http.StatusNotFound, "Page not found")
		}

	default:
		h.Logger.ErrorContext(r.Context(), "unsupported method in device callback", "method", r.Method)
		h.RenderError(r, w, http.StatusBadRequest, "Method not allowed.")
		return
	}
}

func (h *Handler) handleDeviceTokenDeprecated(w http.ResponseWriter, r *http.Request) {
	h.Logger.Warn(`the /device/token endpoint was called. It will be removed, use /token instead.`, "deprecated", true)

	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			h.Logger.Warn("could not parse Device Token Request body", "err", err)
			h.writeError(w, oauth2.InvalidRequest, "", http.StatusBadRequest)
			return
		}

		grantType := r.PostFormValue("grant_type")
		if grantType != oauth2.GrantTypeDeviceCode {
			h.writeError(w, oauth2.InvalidGrant, "", http.StatusBadRequest)
			return
		}

		h.HandleToken(w, r)
	default:
		h.RenderError(r, w, http.StatusBadRequest, "Requested resource does not exist.")
	}
}

// HandleToken serves the device_code grant: it is dispatched from the /token
// endpoint (and the deprecated /device/token endpoint) once the request form has
// been parsed. It polls for the token minted by the browser callback.
func (h *Handler) HandleToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deviceCode := r.Form.Get("device_code")
	if deviceCode == "" {
		h.writeError(w, oauth2.InvalidRequest, "No device code received", http.StatusBadRequest)
		return
	}

	now := h.Now()

	// Grab the device token, check validity
	deviceToken, err := h.Storage.GetDeviceToken(ctx, deviceCode)
	if err != nil {
		if err != storage.ErrNotFound {
			h.Logger.ErrorContext(r.Context(), "failed to get device code", "err", err)
		}
		h.writeError(w, oauth2.InvalidRequest, "Invalid Device code.", http.StatusBadRequest)
		return
	} else if now.After(deviceToken.Expiry) {
		h.writeError(w, oauth2.DeviceTokenExpired, "", http.StatusBadRequest)
		return
	}

	// Rate Limiting check
	slowDown := false
	pollInterval := deviceToken.PollIntervalSeconds
	minRequestTime := deviceToken.LastRequestTime.Add(time.Second * time.Duration(pollInterval))
	if now.Before(minRequestTime) {
		slowDown = true
		// Continually increase the poll interval until the user waits the proper time
		pollInterval += 5
	} else {
		pollInterval = 5
	}

	switch deviceToken.Status {
	case oauth2.DeviceTokenPending:
		updater := func(old storage.DeviceToken) (storage.DeviceToken, error) {
			old.PollIntervalSeconds = pollInterval
			old.LastRequestTime = now
			return old, nil
		}
		// Update device token last request time in storage
		if err := h.Storage.UpdateDeviceToken(ctx, deviceCode, updater); err != nil {
			h.Logger.ErrorContext(r.Context(), "failed to update device token", "err", err)
			h.RenderError(r, w, http.StatusInternalServerError, "")
			return
		}
		if slowDown {
			h.writeError(w, oauth2.DeviceTokenSlowDown, "", http.StatusBadRequest)
		} else {
			h.writeError(w, oauth2.DeviceTokenPending, "", http.StatusBadRequest)
		}
	case oauth2.DeviceTokenComplete:
		codeChallengeFromStorage := deviceToken.PKCE.CodeChallenge
		providedCodeVerifier := r.Form.Get("code_verifier")

		switch {
		case providedCodeVerifier != "" && codeChallengeFromStorage != "":
			calculatedCodeChallenge, err := h.CalculateCodeChallenge(providedCodeVerifier, deviceToken.PKCE.CodeChallengeMethod)
			if err != nil {
				h.Logger.ErrorContext(r.Context(), "failed to calculate code challenge", "err", err)
				h.writeError(w, oauth2.ServerError, "", http.StatusInternalServerError)
				return
			}
			if codeChallengeFromStorage != calculatedCodeChallenge {
				h.writeError(w, oauth2.InvalidGrant, "Invalid code_verifier.", http.StatusBadRequest)
				return
			}
		case providedCodeVerifier != "":
			// Received no code_challenge on /auth, but a code_verifier on /token
			h.writeError(w, oauth2.InvalidRequest, "No PKCE flow started. Cannot check code_verifier.", http.StatusBadRequest)
			return
		case codeChallengeFromStorage != "":
			// Received PKCE request on /auth, but no code_verifier on /token
			h.writeError(w, oauth2.InvalidGrant, "Expecting parameter code_verifier in PKCE flow.", http.StatusBadRequest)
			return
		}
		w.Write([]byte(deviceToken.Token))
	}
}
