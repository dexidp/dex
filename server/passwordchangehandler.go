package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dexidp/dex/storage"
	"golang.org/x/crypto/bcrypt"
)

const passwordChangeURI = "/password/change"

func buildPasswordChangeURI(issuerUrl, username, backURI string, changeReason passwordChangeReason) (string, error) {
	retUrl, err := url.JoinPath(issuerUrl, passwordChangeURI)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"%s?username=%s&back=%s&changeReason=%s",
		retUrl, url.QueryEscape(username), url.QueryEscape(backURI), changeReason,
	), nil
}

func (s *Server) handlePasswordChange(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := r.ParseForm()
	if err != nil {
		s.logger.ErrorContext(ctx, "form parse error", "error", err)
		s.renderError(r, w, http.StatusInternalServerError, "Login error.")
		return
	}

	username := r.Form.Get("username")
	currentPassword := r.Form.Get("currentPassword")
	newPassword := r.Form.Get("newPassword")
	changeReason := r.Form.Get("changeReason")
	backURL := r.Form.Get("back")
	if backURL == "" {
		backURL = s.issuerURL.String()
	}
	
	var passwordHint string
	if s.passwordPolicy != nil {
		passwordHint = s.passwordPolicy.complexity.UserPrompt()
	}

	switch r.Method {
	case http.MethodGet:
		if err := s.templates.passwordChange(r, w, passwordChangeParams{
			Username:        username,
			NewPasswordHint: passwordHint,
			ChangeReason:    passwordChangeReason(changeReason),
			IssuerURL:       s.issuerURL.String(),
		}); err != nil {
			s.logger.ErrorContext(r.Context(), "server template error", "err", err)
		}
		return
	case http.MethodPost:
		if currentPassword == newPassword {
			if err := s.templates.passwordChange(r, w, passwordChangeParams{
				Username:        username,
				NewPasswordHint: passwordHint,
				IssuerURL:       s.issuerURL.String(),
				ChangeReason:    passwordChangeReason(changeReason),
				Err:             ErrOldAndNewPassAreEq,
			}); err != nil {
				s.logger.ErrorContext(r.Context(), "server template error", "err", err)
			}
			return
		}

		if strings.Contains(newPassword, " ") {
			if err := s.templates.passwordChange(r, w, passwordChangeParams{
				Username:        username,
				NewPasswordHint: passwordHint,
				IssuerURL:       s.issuerURL.String(),
				ChangeReason:    passwordChangeReason(changeReason),
				Err:             ErrNewPasswordContainsForbiddenChar,
			}); err != nil {
				s.logger.ErrorContext(r.Context(), "server template error", "err", err)
			}
		}

		if s.passwordPolicy != nil {
			if err := s.passwordPolicy.complexity.Validate(newPassword); err != nil {
				if err := s.templates.passwordChange(r, w, passwordChangeParams{
					Username:        username,
					NewPasswordHint: passwordHint,
					IssuerURL:       s.issuerURL.String(),
					ChangeReason:    passwordChangeReason(changeReason),
					Err:             errors.Join(ErrPasswordTooWeak, err),
				}); err != nil {
					s.logger.ErrorContext(r.Context(), "server template error", "err", err)
				}
				return
			}
		}

		p, err := s.storage.GetPassword(ctx, username)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "failed to get password", "username", username, "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}

		if err := checkCost(p.Hash); err != nil {
			s.logger.ErrorContext(r.Context(), "checkCost failed", "username", username, "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}

		if err := bcrypt.CompareHashAndPassword(p.Hash, []byte(currentPassword)); err != nil {
			if err := s.templates.passwordChange(r, w, passwordChangeParams{
				Username:        username,
				NewPasswordHint: passwordHint,
				IssuerURL:       s.issuerURL.String(),
				ChangeReason:    passwordChangeReason(changeReason),
				Err:             ErrCurrentPasswordInvalid,
			}); err != nil {
				s.logger.ErrorContext(r.Context(), "server template error", "err", err)
			}
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
		if err != nil {
			s.logger.ErrorContext(ctx, "bcrypt.GenerateFromPassword", "username", username, "error", err)
			s.renderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}

		for _, el := range p.PreviousHashes {
			if err := bcrypt.CompareHashAndPassword(el, []byte(newPassword)); err == nil {
				if err := s.templates.passwordChange(r, w, passwordChangeParams{
					Username:        username,
					NewPasswordHint: passwordHint,
					IssuerURL:       s.issuerURL.String(),
					ChangeReason:    passwordChangeReason(changeReason),
					Err:             ErrReusedPassword,
				}); err != nil {
					s.logger.ErrorContext(r.Context(), "server template error", "err", err)
				}
				return
			}
		}

		updater := func(p storage.Password) (storage.Password, error) {
			p.PreviousHashes = append(p.PreviousHashes, p.Hash)
			if len(p.PreviousHashes) > int(s.passwordPolicy.passwordHistoryLimit) {
				p.PreviousHashes = p.PreviousHashes[1:]
			}
			p.HashUpdatedAt = time.Now()
			p.Hash = hash
			if s.passwordPolicy != nil {
				p.ComplexityLevel = s.passwordPolicy.complexity.level.String()
			}
			return p, nil
		}
		if err := s.storage.UpdatePassword(ctx, username, updater); err != nil {
			s.logger.ErrorContext(r.Context(), "failed to update password", "username", username, "err", err)
			s.renderError(r, w, http.StatusInternalServerError, "Login error.")
			return
		}

		s.logger.InfoContext(r.Context(), "password changed successfully", "username", username)

		http.Redirect(w, r, backURL, http.StatusSeeOther)
	default:
		s.renderError(r, w, http.StatusBadRequest, "Method not supported")
	}
}
