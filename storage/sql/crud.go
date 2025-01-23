package sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dexidp/dex/storage"
)

// TODO(ericchiang): The update, insert, and select methods queries are all
// very repetitive. Consider creating them programmatically.

// keysRowID is the ID of the only row we expect to populate the "keys" table.
const keysRowID = "keys"

// encoder wraps the underlying value in a JSON marshaler which is automatically
// called by the database/sql package.
//
//	s := []string{"planes", "bears"}
//	err := db.Exec(`insert into t1 (id, things) values (1, $1)`, encoder(s))
//	if err != nil {
//		// handle error
//	}
//
//	var r []byte
//	err = db.QueryRow(`select things from t1 where id = 1;`).Scan(&r)
//	if err != nil {
//		// handle error
//	}
//	fmt.Printf("%s\n", r) // ["planes","bears"]
func encoder(i interface{}) driver.Valuer {
	return jsonEncoder{i}
}

// decoder wraps the underlying value in a JSON unmarshaler which can then be passed
// to a database Scan() method.
func decoder(i interface{}) sql.Scanner {
	return jsonDecoder{i}
}

type jsonEncoder struct {
	i interface{}
}

func (j jsonEncoder) Value() (driver.Value, error) {
	b, err := json.Marshal(j.i)
	if err != nil {
		return nil, fmt.Errorf("marshal: %v", err)
	}
	return b, nil
}

type jsonDecoder struct {
	i interface{}
}

func (j jsonDecoder) Scan(dest interface{}) error {
	if dest == nil {
		return errors.New("nil value")
	}
	b, ok := dest.([]byte)
	if !ok {
		return fmt.Errorf("expected []byte got %T", dest)
	}
	if err := json.Unmarshal(b, &j.i); err != nil {
		return fmt.Errorf("unmarshal: %v", err)
	}
	return nil
}

// Abstract conn vs trans.
type querier interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

// Abstract row vs rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

var _ storage.Storage = (*conn)(nil)

func (c *conn) GarbageCollect(ctc context.Context, now time.Time) (storage.GCResult, error) {
	result := storage.GCResult{}

	r, err := c.Exec(`delete from auth_request where expiry < $1`, now)
	if err != nil {
		return result, fmt.Errorf("gc auth_request: %v", err)
	}
	if n, err := r.RowsAffected(); err == nil {
		result.AuthRequests = n
	}

	r, err = c.Exec(`delete from auth_code where expiry < $1`, now)
	if err != nil {
		return result, fmt.Errorf("gc auth_code: %v", err)
	}
	if n, err := r.RowsAffected(); err == nil {
		result.AuthCodes = n
	}

	r, err = c.Exec(`delete from device_request where expiry < $1`, now)
	if err != nil {
		return result, fmt.Errorf("gc device_request: %v", err)
	}
	if n, err := r.RowsAffected(); err == nil {
		result.DeviceRequests = n
	}

	r, err = c.Exec(`delete from device_token where expiry < $1`, now)
	if err != nil {
		return result, fmt.Errorf("gc device_token: %v", err)
	}
	if n, err := r.RowsAffected(); err == nil {
		result.DeviceTokens = n
	}

	return result, err
}

func (c *conn) CreateAuthRequest(ctx context.Context, a storage.AuthRequest) error {
	_, err := c.Exec(`
		insert into auth_request (
			id, client_id, response_types, scopes, redirect_uri, nonce, state,
			force_approval_prompt, logged_in,
			claims_user_id, claims_username, claims_preferred_username,
			claims_email, claims_email_verified, claims_groups,
			connector_id, connector_data,
			expiry,
			code_challenge, code_challenge_method,
			hmac_key
		)
		values (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		);
	`,
		a.ID, a.ClientID, encoder(a.ResponseTypes), encoder(a.Scopes), a.RedirectURI, a.Nonce, a.State,
		a.ForceApprovalPrompt, a.LoggedIn,
		a.Claims.UserID, a.Claims.Username, a.Claims.PreferredUsername,
		a.Claims.Email, a.Claims.EmailVerified, encoder(a.Claims.Groups),
		a.ConnectorID, a.ConnectorData,
		a.Expiry,
		a.PKCE.CodeChallenge, a.PKCE.CodeChallengeMethod,
		a.HMACKey,
	)
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert auth request: %v", err)
	}
	return nil
}

func (c *conn) UpdateAuthRequest(ctx context.Context, id string, updater func(a storage.AuthRequest) (storage.AuthRequest, error)) error {
	return c.ExecTx(func(tx *trans) error {
		r, err := getAuthRequest(ctx, tx, id)
		if err != nil {
			return err
		}

		a, err := updater(r)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`
			update auth_request
			set
				client_id = $1, response_types = $2, scopes = $3, redirect_uri = $4,
				nonce = $5, state = $6, force_approval_prompt = $7, logged_in = $8,
				claims_user_id = $9, claims_username = $10, claims_preferred_username = $11,
				claims_email = $12, claims_email_verified = $13,
				claims_groups = $14,
				connector_id = $15, connector_data = $16,
				expiry = $17,
				code_challenge = $18, code_challenge_method = $19,
				hmac_key = $20
			where id = $21;
		`,
			a.ClientID, encoder(a.ResponseTypes), encoder(a.Scopes), a.RedirectURI, a.Nonce, a.State,
			a.ForceApprovalPrompt, a.LoggedIn,
			a.Claims.UserID, a.Claims.Username, a.Claims.PreferredUsername,
			a.Claims.Email, a.Claims.EmailVerified,
			encoder(a.Claims.Groups),
			a.ConnectorID, a.ConnectorData,
			a.Expiry,
			a.PKCE.CodeChallenge, a.PKCE.CodeChallengeMethod, a.HMACKey,
			r.ID,
		)
		if err != nil {
			return fmt.Errorf("update auth request: %v", err)
		}
		return nil
	})
}

func (c *conn) GetAuthRequest(ctx context.Context, id string) (storage.AuthRequest, error) {
	return getAuthRequest(ctx, c, id)
}

func getAuthRequest(ctx context.Context, q querier, id string) (a storage.AuthRequest, err error) {
	err = q.QueryRow(`
		select
			id, client_id, response_types, scopes, redirect_uri, nonce, state,
			force_approval_prompt, logged_in,
			claims_user_id, claims_username, claims_preferred_username,
			claims_email, claims_email_verified, claims_groups,
			connector_id, connector_data, expiry,
			code_challenge, code_challenge_method, hmac_key
		from auth_request where id = $1;
	`, id).Scan(
		&a.ID, &a.ClientID, decoder(&a.ResponseTypes), decoder(&a.Scopes), &a.RedirectURI, &a.Nonce, &a.State,
		&a.ForceApprovalPrompt, &a.LoggedIn,
		&a.Claims.UserID, &a.Claims.Username, &a.Claims.PreferredUsername,
		&a.Claims.Email, &a.Claims.EmailVerified,
		decoder(&a.Claims.Groups),
		&a.ConnectorID, &a.ConnectorData, &a.Expiry,
		&a.PKCE.CodeChallenge, &a.PKCE.CodeChallengeMethod, &a.HMACKey,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return a, storage.ErrNotFound
		}
		return a, fmt.Errorf("select auth request: %v", err)
	}
	return a, nil
}

func (c *conn) CreateAuthCode(ctx context.Context, a storage.AuthCode) error {
	_, err := c.Exec(`
		insert into auth_code (
			id, client_id, scopes, nonce, redirect_uri,
			claims_user_id, claims_username, claims_preferred_username,
			claims_email, claims_email_verified, claims_groups,
			connector_id, connector_data,
			expiry,
			code_challenge, code_challenge_method
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16);
	`,
		a.ID, a.ClientID, encoder(a.Scopes), a.Nonce, a.RedirectURI, a.Claims.UserID,
		a.Claims.Username, a.Claims.PreferredUsername, a.Claims.Email, a.Claims.EmailVerified,
		encoder(a.Claims.Groups), a.ConnectorID, a.ConnectorData, a.Expiry,
		a.PKCE.CodeChallenge, a.PKCE.CodeChallengeMethod,
	)
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert auth code: %v", err)
	}
	return nil
}

func (c *conn) GetAuthCode(ctx context.Context, id string) (a storage.AuthCode, err error) {
	err = c.QueryRow(`
		select
			id, client_id, scopes, nonce, redirect_uri,
			claims_user_id, claims_username, claims_preferred_username,
			claims_email, claims_email_verified, claims_groups,
			connector_id, connector_data,
			expiry,
			code_challenge, code_challenge_method
		from auth_code where id = $1;
	`, id).Scan(
		&a.ID, &a.ClientID, decoder(&a.Scopes), &a.Nonce, &a.RedirectURI, &a.Claims.UserID,
		&a.Claims.Username, &a.Claims.PreferredUsername, &a.Claims.Email, &a.Claims.EmailVerified,
		decoder(&a.Claims.Groups), &a.ConnectorID, &a.ConnectorData, &a.Expiry,
		&a.PKCE.CodeChallenge, &a.PKCE.CodeChallengeMethod,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return a, storage.ErrNotFound
		}
		return a, fmt.Errorf("select auth code: %v", err)
	}
	return a, nil
}

func (c *conn) CreateRefresh(ctx context.Context, r storage.RefreshToken) error {
	_, err := c.Exec(`
		insert into refresh_token (
			id, client_id, scopes, nonce,
			claims_user_id, claims_username, claims_preferred_username,
			claims_email, claims_email_verified, claims_groups,
			connector_id, connector_data,
			token, obsolete_token, created_at, last_used
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16);
	`,
		r.ID, r.ClientID, encoder(r.Scopes), r.Nonce,
		r.Claims.UserID, r.Claims.Username, r.Claims.PreferredUsername,
		r.Claims.Email, r.Claims.EmailVerified,
		encoder(r.Claims.Groups),
		r.ConnectorID, r.ConnectorData,
		r.Token, r.ObsoleteToken, r.CreatedAt, r.LastUsed,
	)
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert refresh_token: %v", err)
	}
	return nil
}

func (c *conn) UpdateRefreshToken(ctx context.Context, id string, updater func(old storage.RefreshToken) (storage.RefreshToken, error)) error {
	return c.ExecTx(func(tx *trans) error {
		r, err := getRefresh(ctx, tx, id)
		if err != nil {
			return err
		}
		if r, err = updater(r); err != nil {
			return err
		}
		_, err = tx.Exec(`
			update refresh_token
			set
				client_id = $1,
				scopes = $2,
				nonce = $3,
				claims_user_id = $4,
				claims_username = $5,
				claims_preferred_username = $6,
				claims_email = $7,
				claims_email_verified = $8,
				claims_groups = $9,
				connector_id = $10,
				connector_data = $11,
				token = $12,
                obsolete_token = $13,
				created_at = $14,
				last_used = $15
			where
				id = $16
		`,
			r.ClientID, encoder(r.Scopes), r.Nonce,
			r.Claims.UserID, r.Claims.Username, r.Claims.PreferredUsername,
			r.Claims.Email, r.Claims.EmailVerified,
			encoder(r.Claims.Groups),
			r.ConnectorID, r.ConnectorData,
			r.Token, r.ObsoleteToken, r.CreatedAt, r.LastUsed, id,
		)
		if err != nil {
			return fmt.Errorf("update refresh token: %v", err)
		}
		return nil
	})
}

func (c *conn) GetRefresh(ctx context.Context, id string) (storage.RefreshToken, error) {
	return getRefresh(ctx, c, id)
}

func getRefresh(ctx context.Context, q querier, id string) (storage.RefreshToken, error) {
	return scanRefresh(q.QueryRow(`
		select
			id, client_id, scopes, nonce,
			claims_user_id, claims_username, claims_preferred_username,
			claims_email, claims_email_verified,
			claims_groups,
			connector_id, connector_data,
			token, obsolete_token, created_at, last_used
		from refresh_token where id = $1;
	`, id))
}

func (c *conn) ListRefreshTokens(ctx context.Context) ([]storage.RefreshToken, error) {
	rows, err := c.Query(`
		select
			id, client_id, scopes, nonce,
			claims_user_id, claims_username, claims_preferred_username,
			claims_email, claims_email_verified, claims_groups,
			connector_id, connector_data,
			token, obsolete_token, created_at, last_used
		from refresh_token;
	`)
	if err != nil {
		return nil, fmt.Errorf("query: %v", err)
	}
	defer rows.Close()

	var tokens []storage.RefreshToken
	for rows.Next() {
		r, err := scanRefresh(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan: %v", err)
	}
	return tokens, nil
}

func scanRefresh(s scanner) (r storage.RefreshToken, err error) {
	err = s.Scan(
		&r.ID, &r.ClientID, decoder(&r.Scopes), &r.Nonce,
		&r.Claims.UserID, &r.Claims.Username, &r.Claims.PreferredUsername,
		&r.Claims.Email, &r.Claims.EmailVerified,
		decoder(&r.Claims.Groups),
		&r.ConnectorID, &r.ConnectorData,
		&r.Token, &r.ObsoleteToken, &r.CreatedAt, &r.LastUsed,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return r, storage.ErrNotFound
		}
		return r, fmt.Errorf("scan refresh_token: %v", err)
	}
	return r, nil
}

func (c *conn) UpdateKeys(ctx context.Context, updater func(old storage.Keys) (storage.Keys, error)) error {
	return c.ExecTx(func(tx *trans) error {
		firstUpdate := false
		// TODO(ericchiang): errors may cause a transaction be rolled back by the SQL
		// server. Test this, and consider adding a COUNT() command beforehand.
		old, err := getKeys(ctx, tx)
		if err != nil {
			if err != storage.ErrNotFound {
				return fmt.Errorf("get keys: %v", err)
			}
			firstUpdate = true
			old = storage.Keys{}
		}

		nk, err := updater(old)
		if err != nil {
			return err
		}

		if firstUpdate {
			_, err = tx.Exec(`
				insert into keys (
					id, verification_keys, signing_key, signing_key_pub, next_rotation
				)
				values ($1, $2, $3, $4, $5);
			`,
				keysRowID, encoder(nk.VerificationKeys), encoder(nk.SigningKey),
				encoder(nk.SigningKeyPub), nk.NextRotation,
			)
			if err != nil {
				return fmt.Errorf("insert: %v", err)
			}
		} else {
			_, err = tx.Exec(`
				update keys
				set
				    verification_keys = $1,
					signing_key = $2,
					signing_key_pub = $3,
					next_rotation = $4
				where id = $5;
			`,
				encoder(nk.VerificationKeys), encoder(nk.SigningKey),
				encoder(nk.SigningKeyPub), nk.NextRotation, keysRowID,
			)
			if err != nil {
				return fmt.Errorf("update: %v", err)
			}
		}
		return nil
	})
}

func (c *conn) GetKeys(ctx context.Context) (keys storage.Keys, err error) {
	return getKeys(ctx, c)
}

func getKeys(ctx context.Context, q querier) (keys storage.Keys, err error) {
	err = q.QueryRow(`
		select
			verification_keys, signing_key, signing_key_pub, next_rotation
		from keys
		where id=$1
	`, keysRowID).Scan(
		decoder(&keys.VerificationKeys), decoder(&keys.SigningKey),
		decoder(&keys.SigningKeyPub), &keys.NextRotation,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return keys, storage.ErrNotFound
		}
		return keys, fmt.Errorf("query keys: %v", err)
	}
	return keys, nil
}

func (c *conn) UpdateClient(ctx context.Context, id string, updater func(old storage.Client) (storage.Client, error)) error {
	return c.ExecTx(func(tx *trans) error {
		cli, err := getClient(ctx, tx, id)
		if err != nil {
			return err
		}
		nc, err := updater(cli)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			update client
			set
				secret = $1,
				redirect_uris = $2,
				trusted_peers = $3,
				public = $4,
				name = $5,
				logo_url = $6
			where id = $7;
		`, nc.Secret, encoder(nc.RedirectURIs), encoder(nc.TrustedPeers), nc.Public, nc.Name, nc.LogoURL, id,
		)
		if err != nil {
			return fmt.Errorf("update client: %v", err)
		}
		return nil
	})
}

func (c *conn) CreateClient(ctx context.Context, cli storage.Client) error {
	_, err := c.Exec(`
		insert into client (
			id, secret, redirect_uris, trusted_peers, public, name, logo_url
		)
		values ($1, $2, $3, $4, $5, $6, $7);
	`,
		cli.ID, cli.Secret, encoder(cli.RedirectURIs), encoder(cli.TrustedPeers),
		cli.Public, cli.Name, cli.LogoURL,
	)
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert client: %v", err)
	}
	return nil
}

func getClient(ctx context.Context, q querier, id string) (storage.Client, error) {
	return scanClient(q.QueryRow(`
		select
			id, secret, redirect_uris, trusted_peers, public, name, logo_url
	    from client where id = $1;
	`, id))
}

func (c *conn) GetClient(ctx context.Context, id string) (storage.Client, error) {
	return getClient(ctx, c, id)
}

func (c *conn) ListClients(ctx context.Context) ([]storage.Client, error) {
	rows, err := c.Query(`
		select
			id, secret, redirect_uris, trusted_peers, public, name, logo_url
		from client;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []storage.Client
	for rows.Next() {
		cli, err := scanClient(rows)
		if err != nil {
			return nil, err
		}
		clients = append(clients, cli)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return clients, nil
}

func scanClient(s scanner) (cli storage.Client, err error) {
	err = s.Scan(
		&cli.ID, &cli.Secret, decoder(&cli.RedirectURIs), decoder(&cli.TrustedPeers),
		&cli.Public, &cli.Name, &cli.LogoURL,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return cli, storage.ErrNotFound
		}
		return cli, fmt.Errorf("get client: %v", err)
	}
	return cli, nil
}

func (c *conn) CreatePassword(ctx context.Context, p storage.Password) error {
	p.Email = strings.ToLower(p.Email)
	_, err := c.Exec(`
		insert into password (
			email, hash, username, user_id
		)
		values (
			$1, $2, $3, $4
		);
	`,
		p.Email, p.Hash, p.Username, p.UserID,
	)
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert password: %v", err)
	}
	return nil
}

func (c *conn) UpdatePassword(ctx context.Context, email string, updater func(p storage.Password) (storage.Password, error)) error {
	return c.ExecTx(func(tx *trans) error {
		p, err := getPassword(ctx, tx, email)
		if err != nil {
			return err
		}

		np, err := updater(p)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`
			update password
			set
				hash = $1, username = $2, user_id = $3
			where email = $4;
		`,
			np.Hash, np.Username, np.UserID, p.Email,
		)
		if err != nil {
			return fmt.Errorf("update password: %v", err)
		}
		return nil
	})
}

func (c *conn) GetPassword(ctx context.Context, email string) (storage.Password, error) {
	return getPassword(ctx, c, email)
}

func getPassword(ctx context.Context, q querier, email string) (p storage.Password, err error) {
	return scanPassword(q.QueryRow(`
		select
			email, hash, username, user_id
		from password where email = $1;
	`, strings.ToLower(email)))
}

func (c *conn) ListPasswords(ctx context.Context) ([]storage.Password, error) {
	rows, err := c.Query(`
		select
			email, hash, username, user_id
		from password;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var passwords []storage.Password
	for rows.Next() {
		p, err := scanPassword(rows)
		if err != nil {
			return nil, err
		}
		passwords = append(passwords, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return passwords, nil
}

func scanPassword(s scanner) (p storage.Password, err error) {
	err = s.Scan(
		&p.Email, &p.Hash, &p.Username, &p.UserID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return p, storage.ErrNotFound
		}
		return p, fmt.Errorf("select password: %v", err)
	}
	return p, nil
}

func (c *conn) CreateOfflineSessions(ctx context.Context, s storage.OfflineSessions) error {
	_, err := c.Exec(`
		insert into offline_session (
			user_id, conn_id, refresh, connector_data
		)
		values (
			$1, $2, $3, $4
		);
	`,
		s.UserID, s.ConnID, encoder(s.Refresh), s.ConnectorData,
	)
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert offline session: %v", err)
	}
	return nil
}

func (c *conn) UpdateOfflineSessions(ctx context.Context, userID string, connID string, updater func(s storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	return c.ExecTx(func(tx *trans) error {
		s, err := getOfflineSessions(ctx, tx, userID, connID)
		if err != nil {
			return err
		}

		newSession, err := updater(s)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`
			update offline_session
			set
				refresh = $1,
				connector_data = $2
			where user_id = $3 AND conn_id = $4;
		`,
			encoder(newSession.Refresh), newSession.ConnectorData, s.UserID, s.ConnID,
		)
		if err != nil {
			return fmt.Errorf("update offline session: %v", err)
		}
		return nil
	})
}

func (c *conn) GetOfflineSessions(ctx context.Context, userID string, connID string) (storage.OfflineSessions, error) {
	return getOfflineSessions(ctx, c, userID, connID)
}

func getOfflineSessions(ctx context.Context, q querier, userID string, connID string) (storage.OfflineSessions, error) {
	return scanOfflineSessions(q.QueryRow(`
		select
			user_id, conn_id, refresh, connector_data
		from offline_session
		where user_id = $1 AND conn_id = $2;
		`, userID, connID))
}

func scanOfflineSessions(s scanner) (o storage.OfflineSessions, err error) {
	err = s.Scan(
		&o.UserID, &o.ConnID, decoder(&o.Refresh), &o.ConnectorData,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return o, storage.ErrNotFound
		}
		return o, fmt.Errorf("select offline session: %v", err)
	}
	return o, nil
}

func (c *conn) CreateConnector(ctx context.Context, connector storage.Connector) error {
	_, err := c.Exec(`
		insert into connector (
			id, type, name, resource_version, config
		)
		values (
			$1, $2, $3, $4, $5
		);
	`,
		connector.ID, connector.Type, connector.Name, connector.ResourceVersion, connector.Config,
	)
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert connector: %v", err)
	}
	return nil
}

func (c *conn) UpdateConnector(ctx context.Context, id string, updater func(s storage.Connector) (storage.Connector, error)) error {
	return c.ExecTx(func(tx *trans) error {
		connector, err := getConnector(ctx, tx, id)
		if err != nil {
			return err
		}

		newConn, err := updater(connector)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`
			update connector
			set
			    type = $1,
			    name = $2,
			    resource_version = $3,
			    config = $4
			where id = $5;
		`,
			newConn.Type, newConn.Name, newConn.ResourceVersion, newConn.Config, connector.ID,
		)
		if err != nil {
			return fmt.Errorf("update connector: %v", err)
		}
		return nil
	})
}

func (c *conn) GetConnector(ctx context.Context, id string) (storage.Connector, error) {
	return getConnector(ctx, c, id)
}

func getConnector(ctx context.Context, q querier, id string) (storage.Connector, error) {
	return scanConnector(q.QueryRow(`
		select
			id, type, name, resource_version, config
		from connector
		where id = $1;
		`, id))
}

func scanConnector(s scanner) (c storage.Connector, err error) {
	err = s.Scan(
		&c.ID, &c.Type, &c.Name, &c.ResourceVersion, &c.Config,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return c, storage.ErrNotFound
		}
		return c, fmt.Errorf("select connector: %v", err)
	}
	return c, nil
}

func (c *conn) ListConnectors(ctx context.Context) ([]storage.Connector, error) {
	rows, err := c.Query(`
		select
			id, type, name, resource_version, config
		from connector;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connectors []storage.Connector
	for rows.Next() {
		conn, err := scanConnector(rows)
		if err != nil {
			return nil, err
		}
		connectors = append(connectors, conn)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return connectors, nil
}

func (c *conn) DeleteAuthRequest(ctx context.Context, id string) error {
	return c.delete("auth_request", "id", id)
}

func (c *conn) DeleteAuthCode(ctx context.Context, id string) error {
	return c.delete("auth_code", "id", id)
}

func (c *conn) DeleteClient(ctx context.Context, id string) error {
	return c.delete("client", "id", id)
}

func (c *conn) DeleteRefresh(ctx context.Context, id string) error {
	return c.delete("refresh_token", "id", id)
}

func (c *conn) DeletePassword(ctx context.Context, email string) error {
	return c.delete("password", "email", strings.ToLower(email))
}

func (c *conn) DeleteConnector(ctx context.Context, id string) error {
	return c.delete("connector", "id", id)
}

func (c *conn) DeleteOfflineSessions(ctx context.Context, userID string, connID string) error {
	result, err := c.Exec(`delete from offline_session where user_id = $1 AND conn_id = $2`, userID, connID)
	if err != nil {
		return fmt.Errorf("delete offline_session: user_id = %s, conn_id = %s", userID, connID)
	}

	// For now mandate that the driver implements RowsAffected. If we ever need to support
	// a driver that doesn't implement this, we can run this in a transaction with a get beforehand.
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %v", err)
	}
	if n < 1 {
		return storage.ErrNotFound
	}
	return nil
}

// Do NOT call directly. Does not escape table.
func (c *conn) delete(table, field, id string) error {
	result, err := c.Exec(`delete from `+table+` where `+field+` = $1`, id)
	if err != nil {
		return fmt.Errorf("delete %s: %v", table, id)
	}

	// For now mandate that the driver implements RowsAffected. If we ever need to support
	// a driver that doesn't implement this, we can run this in a transaction with a get beforehand.
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %v", err)
	}
	if n < 1 {
		return storage.ErrNotFound
	}
	return nil
}

func (c *conn) CreateDeviceRequest(ctx context.Context, d storage.DeviceRequest) error {
	_, err := c.Exec(`
		insert into device_request (
			user_code, device_code, client_id, client_secret, scopes, expiry
		)
		values (
			$1, $2, $3, $4, $5, $6
		);`,
		d.UserCode, d.DeviceCode, d.ClientID, d.ClientSecret, encoder(d.Scopes), d.Expiry,
	)
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert device request: %v", err)
	}
	return nil
}

func (c *conn) CreateDeviceToken(ctx context.Context, t storage.DeviceToken) error {
	_, err := c.Exec(`
		insert into device_token (
			device_code, status, token, expiry, last_request, poll_interval, code_challenge, code_challenge_method
		)
		values (
			$1, $2, $3, $4, $5, $6, $7, $8
		);`,
		t.DeviceCode, t.Status, t.Token, t.Expiry, t.LastRequestTime, t.PollIntervalSeconds, t.PKCE.CodeChallenge, t.PKCE.CodeChallengeMethod,
	)
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert device token: %v", err)
	}
	return nil
}

func (c *conn) GetDeviceRequest(ctx context.Context, userCode string) (storage.DeviceRequest, error) {
	return getDeviceRequest(ctx, c, userCode)
}

func getDeviceRequest(ctx context.Context, q querier, userCode string) (d storage.DeviceRequest, err error) {
	err = q.QueryRow(`
		select
            device_code, client_id, client_secret, scopes, expiry
		from device_request where user_code = $1;
	`, userCode).Scan(
		&d.DeviceCode, &d.ClientID, &d.ClientSecret, decoder(&d.Scopes), &d.Expiry,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return d, storage.ErrNotFound
		}
		return d, fmt.Errorf("select device token: %v", err)
	}
	d.UserCode = userCode
	return d, nil
}

func (c *conn) GetDeviceToken(ctx context.Context, deviceCode string) (storage.DeviceToken, error) {
	return getDeviceToken(ctx, c, deviceCode)
}

func getDeviceToken(ctx context.Context, q querier, deviceCode string) (a storage.DeviceToken, err error) {
	err = q.QueryRow(`
		select
            status, token, expiry, last_request, poll_interval, code_challenge, code_challenge_method
		from device_token where device_code = $1;
	`, deviceCode).Scan(
		&a.Status, &a.Token, &a.Expiry, &a.LastRequestTime, &a.PollIntervalSeconds, &a.PKCE.CodeChallenge, &a.PKCE.CodeChallengeMethod,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return a, storage.ErrNotFound
		}
		return a, fmt.Errorf("select device token: %v", err)
	}
	a.DeviceCode = deviceCode
	return a, nil
}

func (c *conn) UpdateDeviceToken(ctx context.Context, deviceCode string, updater func(old storage.DeviceToken) (storage.DeviceToken, error)) error {
	return c.ExecTx(func(tx *trans) error {
		r, err := getDeviceToken(ctx, tx, deviceCode)
		if err != nil {
			return err
		}
		if r, err = updater(r); err != nil {
			return err
		}
		_, err = tx.Exec(`
			update device_token
			set
				status = $1,
				token = $2,
				last_request = $3,
				poll_interval = $4,
				code_challenge = $5,
				code_challenge_method = $6
			where
				device_code = $7
		`,
			r.Status, r.Token, r.LastRequestTime, r.PollIntervalSeconds, r.PKCE.CodeChallenge, r.PKCE.CodeChallengeMethod, r.DeviceCode,
		)
		if err != nil {
			return fmt.Errorf("update device token: %v", err)
		}
		return nil
	})
}
