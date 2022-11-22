package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dexidp/dex/api/v2"
	i "github.com/dexidp/dex/server/internal"
)

type vehicleIDTokenClaims struct {
	Issuer   string   `json:"iss"`
	Subject  string   `json:"sub"`
	Audience audience `json:"aud,omitempty"`
	Expiry   int64    `json:"exp"`
	IssuedAt int64    `json:"iat"`
	UserID   string   `json:"userId"`

	Privileges []int64 `json:"privileges,omitempty"`
}

func (d dexAPI) GetVehiclePrivilegeToken(ctx context.Context, req *api.GetVehiclePrivilegeTokenReq) (*api.GetVehiclePrivilegeTokenResp, error) {
	j, err := i.NewJwtHelper(d.s, d.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Jwt token: %v", err)
	}

	expiry := d.serverConfig.Now().Add(time.Minute * 10) // TODO - Discuss an appropriate time
	v := vehicleIDTokenClaims{
		Issuer:     d.serverConfig.Issuer,
		Subject:    req.VehicleTokenId,
		Expiry:     expiry.Unix(),
		IssuedAt:   d.serverConfig.Now().Unix(),
		Privileges: req.PrivilegeIds,
		UserID:     req.UserId,
	}

	payload, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("could not serialize claims: %v", err)
	}

	token, err := j.SignPayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Jwt token: %v", err)
	}

	return &api.GetVehiclePrivilegeTokenResp{
		Token: token,
	}, nil
}
