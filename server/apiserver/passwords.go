package apiserver

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/server/passwords"
	"github.com/dexidp/dex/storage"
)

func (d dexAPI) CreatePassword(ctx context.Context, req *api.CreatePasswordReq) (*api.CreatePasswordResp, error) {
	if req.Password == nil {
		return nil, errors.New("no password supplied")
	}
	if req.Password.UserId == "" {
		return nil, errors.New("no user ID supplied")
	}
	if req.Password.Hash != nil {
		if err := passwords.CheckCost(req.Password.Hash); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("no hash of password supplied")
	}

	p := storage.Password{
		Email:    req.Password.Email,
		Hash:     req.Password.Hash,
		Username: req.Password.Username,
		UserID:   req.Password.UserId,
	}
	if err := d.s.CreatePassword(ctx, p); err != nil {
		if err == storage.ErrAlreadyExists {
			return &api.CreatePasswordResp{AlreadyExists: true}, nil
		}
		d.logger.Error("failed to create password", "err", err)
		return nil, fmt.Errorf("create password: %v", err)
	}

	return &api.CreatePasswordResp{}, nil
}

func (d dexAPI) UpdatePassword(ctx context.Context, req *api.UpdatePasswordReq) (*api.UpdatePasswordResp, error) {
	if req.Email == "" {
		return nil, errors.New("no email supplied")
	}
	if req.NewHash == nil && req.NewUsername == "" {
		return nil, errors.New("nothing to update")
	}

	if req.NewHash != nil {
		if err := passwords.CheckCost(req.NewHash); err != nil {
			return nil, err
		}
	}

	updater := func(old storage.Password) (storage.Password, error) {
		if req.NewHash != nil {
			old.Hash = req.NewHash
		}

		if req.NewUsername != "" {
			old.Username = req.NewUsername
		}

		return old, nil
	}

	if err := d.s.UpdatePassword(ctx, req.Email, updater); err != nil {
		if err == storage.ErrNotFound {
			return &api.UpdatePasswordResp{NotFound: true}, nil
		}
		d.logger.Error("failed to update password", "err", err)
		return nil, fmt.Errorf("update password: %v", err)
	}

	return &api.UpdatePasswordResp{}, nil
}

func (d dexAPI) DeletePassword(ctx context.Context, req *api.DeletePasswordReq) (*api.DeletePasswordResp, error) {
	if req.Email == "" {
		return nil, errors.New("no email supplied")
	}

	err := d.s.DeletePassword(ctx, req.Email)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.DeletePasswordResp{NotFound: true}, nil
		}
		d.logger.Error("failed to delete password", "err", err)
		return nil, fmt.Errorf("delete password: %v", err)
	}
	return &api.DeletePasswordResp{}, nil
}

func (d dexAPI) ListPasswords(ctx context.Context, req *api.ListPasswordReq) (*api.ListPasswordResp, error) {
	passwordList, err := d.s.ListPasswords(ctx)
	if err != nil {
		d.logger.Error("failed to list passwords", "err", err)
		return nil, fmt.Errorf("list passwords: %v", err)
	}

	passwords := make([]*api.Password, 0, len(passwordList))
	for _, password := range passwordList {
		p := api.Password{
			Email:    password.Email,
			Username: password.Username,
			UserId:   password.UserID,
		}
		passwords = append(passwords, &p)
	}

	return &api.ListPasswordResp{
		Passwords: passwords,
	}, nil
}

func (d dexAPI) VerifyPassword(ctx context.Context, req *api.VerifyPasswordReq) (*api.VerifyPasswordResp, error) {
	if req.Email == "" {
		return nil, errors.New("no email supplied")
	}

	if req.Password == "" {
		return nil, errors.New("no password to verify supplied")
	}

	password, err := d.s.GetPassword(ctx, req.Email)
	if err != nil {
		if err == storage.ErrNotFound {
			return &api.VerifyPasswordResp{
				NotFound: true,
			}, nil
		}
		d.logger.Error("there was an error retrieving the password", "err", err)
		return nil, fmt.Errorf("verify password: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword(password.Hash, []byte(req.Password)); err != nil {
		d.logger.Info("password check failed", "err", err)
		return &api.VerifyPasswordResp{
			Verified: false,
		}, nil
	}
	return &api.VerifyPasswordResp{
		Verified: true,
	}, nil
}
