package rememberme

import (
	"context"
	"crypto/sha3"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/internal/jwt"
	"github.com/dexidp/dex/storage"
)

const ACTIVE_SESSION_COOKIE_NAME = "dex_active_session_cookie"

var emptySession = storage.ActiveSession{}

type AuthContext struct {
	connectorName            string
	identity                 *connector.Identity
	configuredExpiryDuration time.Duration
}

func NewAnonymousAuthContext(connectorName string, configuredExpiryDuration time.Duration) AuthContext {
	return AuthContext{connectorName, nil, configuredExpiryDuration}
}

func NewAuthContextWithIdentity(connectorName string, identity connector.Identity, configuredExpiryDuration time.Duration) AuthContext {
	return AuthContext{connectorName, &identity, configuredExpiryDuration}
}

type GetOrUnsetCookie struct {
	cookie *http.Cookie
	unset  bool
}

func (c GetOrUnsetCookie) Empty() bool {
	return c.unset == false && c.cookie == nil
}

func (c GetOrUnsetCookie) Get() (*http.Cookie, bool) {
	// TODO(juf): would prefer to not return internal pointer
	return c.cookie, c.unset
}

func RequestUnsetCookie(cookieName string) GetOrUnsetCookie {
	return GetOrUnsetCookie{
		&http.Cookie{Name: cookieName, Path: "/", MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode}, true,
	}
}

func RequestSetCookie(cookie http.Cookie) GetOrUnsetCookie {
	return GetOrUnsetCookie{
		&cookie, false,
	}
}

type RememberMeCtx struct {
	Session storage.ActiveSession
	Cookie  GetOrUnsetCookie
}

func (ctx RememberMeCtx) IsValid() bool {
	return ctx.Session.Expiry.After(time.Now())
}

// connector_cookie_name creates a string which is used to identify the cookie that matches the given connector.
// The purpose is to avoid having one cookie for multiple providers where you only authenticate once and suddenly would have
// access to other connectors.
func connector_cookie_name(connName string) string {
	return fmt.Sprintf("%s_%s", ACTIVE_SESSION_COOKIE_NAME, connName)
}

// HandleRememberMe either retrieves or creates a Session based on the cookie for the respective connector present in the http.Request.
// It is also responsible for issuing the unsetting / expiration of either an invalid or expired cookie.
//
// The current "design" of the cookie is a sha3 hash of the connector.Identity object as JWK signed payload.
func HandleRememberMe(ctx context.Context, logger *slog.Logger, req *http.Request, data AuthContext, store storage.Storage, sessionStore storage.ActiveSessionStorage) (*RememberMeCtx, error) {
	keys, err := store.GetKeys(ctx)
	if err != nil {
		logger.ErrorContext(req.Context(), "failed to get keys", "err", err)
		return nil, err
	}
	signAlg, err := jwt.SignatureAlgorithm(keys.SigningKey)
	if err != nil {
		logger.ErrorContext(req.Context(), "failed to get signAlg", "err", err)
		return nil, err
	}
	if val, found := extractCookie(req, data.connectorName); found {
		cookieName := connector_cookie_name(data.connectorName)
		logger.DebugContext(req.Context(), "returning user cookie found, checking for active session", "connectorName", data.connectorName)
		keyset := jwt.NewStorageKeySet(store)
		logger.DebugContext(req.Context(), "verifying cookie", "connectorName", data.connectorName)
		_, err := keyset.VerifySignature(ctx, val)
		if err != nil {
			return &RememberMeCtx{
				Session: emptySession,
				Cookie:  RequestUnsetCookie(cookieName),
			}, err
		}
		session, err := sessionStore.GetSession(ctx, val)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return &RememberMeCtx{
					Session: session,
					Cookie:  RequestUnsetCookie(cookieName),
				}, nil
			}
			logger.ErrorContext(req.Context(), "failed to get active session", "err", err, "connectorName", data.connectorName)
			return nil, err
		}
		cookie := GetOrUnsetCookie{nil, false}
		if session.Expiry.Before(time.Now()) {
			logger.DebugContext(req.Context(), "session expired unsetting cookie", "connectorName", data.connectorName)
			cookie = RequestUnsetCookie(cookieName)
		}
		return &RememberMeCtx{
			Session: session,
			Cookie:  cookie,
		}, nil
	} else {
		if data.identity == nil {
			logger.DebugContext(req.Context(), "identity is empty, returning early", "connectorName", data.connectorName)
			return nil, storage.ErrNotFound
		}
		h := sha3.New512()
		h.Write([]byte(data.identity.Email))
		for _, g := range data.identity.Groups {
			h.Write([]byte(g))
		}
		h.Write([]byte(data.identity.UserID))
		h.Write([]byte(data.identity.Username))
		h.Write([]byte(data.identity.PreferredUsername))
		hash := fmt.Sprintf("%x", h.Sum(nil))
		signedHash, err := jwt.SignPayload(keys.SigningKey, signAlg, []byte(hash))
		if err != nil {
			logger.ErrorContext(req.Context(), "failed to get sign payload", "err", err, "connectorName", data.connectorName)
			return nil, err
		}
		// TODO(juf): Double check what we need to persist and are given
		// in the context of whether we need to make an "auto-redirect"
		// Because technically we do not return the ID nor RefreshToken to the user
		// instead we redirect him back to the caller with an authCode
		session := storage.ActiveSession{
			Identity: *data.identity, // TODO(juf): Avoid nil pointer
			// TODO(juf): Think about changing to use Token IssuedAt date instead of now to have
			// alignment with the token
			Expiry: time.Now().Add(data.configuredExpiryDuration),
		}
		logger.DebugContext(req.Context(), "creating active session for user", "connectorName", data.connectorName)
		if err := sessionStore.CreateSession(ctx, signedHash, session); err != nil {
			logger.ErrorContext(req.Context(), "failed to store active session", "err", err, "connectorName", data.connectorName)
			return nil, err
		}

		return &RememberMeCtx{
			Session: session,
			Cookie: RequestSetCookie(http.Cookie{
				Name:     connector_cookie_name(data.connectorName),
				Value:    signedHash,
				Path:     "/",
				Domain:   "", // TODO(juf): Check if we need to set this
				Expires:  session.Expiry,
				MaxAge:   int(time.Until(session.Expiry).Seconds()),
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			}),
		}, nil
	}
}

func extractCookie(req *http.Request, connName string) (value string, found bool) {
	cookies := req.Cookies()
	if len(cookies) > 0 {
		for _, ck := range cookies {
			if ck.Name != connector_cookie_name(connName) {
				continue
			}
			return ck.Value, true
		}
	}
	return "", false
}
