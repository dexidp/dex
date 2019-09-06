package sql

import (
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

const (
	// keysRowID is the ID of the only row we expect to populate the "keys" table.
	keysRowID = "keys"
)

// encoder wraps the underlying value in a JSON marshaler which is automatically
// called by the database/sql package.
//
//		s := []string{"planes", "bears"}
//		err := db.Exec(`insert into t1 (id, things) values (1, $1)`, encoder(s))
//		if err != nil {
//			// handle error
//		}
//
//		var r []byte
//		err = db.QueryRow(`select things from t1 where id = 1;`).Scan(&r)
//		if err != nil {
//			// handle error
//		}
//		fmt.Printf("%s\n", r) // ["planes","bears"]
//
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

func (c *conn) GarbageCollect(now time.Time) (result storage.GCResult, err error) {
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
	return
}

func (c *conn) CreateAuthRequest(a storage.AuthRequest) error {
	_, err := c.Exec(`
		insert into auth_request (
			id, client_id, response_types, scopes, redirect_uri, nonce, state,
			force_approval_prompt, logged_in,
			claims_user_id, claims_username, claims_email, claims_email_verified,
			claims_groups,
			connector_id, connector_data,
			expiry
		)
		values (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		);
	`,
		a.ID, a.ClientID, encoder(a.ResponseTypes), encoder(a.Scopes), a.RedirectURI, a.Nonce, a.State,
		a.ForceApprovalPrompt, a.LoggedIn,
		a.Claims.UserID, a.Claims.Username, a.Claims.Email, a.Claims.EmailVerified,
		encoder(a.Claims.Groups),
		a.ConnectorID, a.ConnectorData,
		a.Expiry,
	)
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.Error{Code: storage.ErrAlreadyExists}
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("insert auth request: %v", err)}
	}
	return nil
}

func (c *conn) UpdateAuthRequest(id string, updater func(a storage.AuthRequest) (storage.AuthRequest, error)) error {
	return c.ExecTx(func(tx *trans) error {
		r, err := getAuthRequest(tx, id)
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
				claims_user_id = $9, claims_username = $10, claims_email = $11,
				claims_email_verified = $12,
				claims_groups = $13,
				connector_id = $14, connector_data = $15,
				expiry = $16
			where id = $17;
		`,
			a.ClientID, encoder(a.ResponseTypes), encoder(a.Scopes), a.RedirectURI, a.Nonce, a.State,
			a.ForceApprovalPrompt, a.LoggedIn,
			a.Claims.UserID, a.Claims.Username, a.Claims.Email, a.Claims.EmailVerified,
			encoder(a.Claims.Groups),
			a.ConnectorID, a.ConnectorData,
			a.Expiry, r.ID,
		)
		if err != nil {
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("update auth request: %v", err)}
		}
		return nil
	})

}

func (c *conn) GetAuthRequest(id string) (storage.AuthRequest, error) {
	return getAuthRequest(c, id)
}

func getAuthRequest(q querier, id string) (a storage.AuthRequest, err error) {
	err = q.QueryRow(`
		select 
			id, client_id, response_types, scopes, redirect_uri, nonce, state,
			force_approval_prompt, logged_in,
			claims_user_id, claims_username, claims_email, claims_email_verified,
			claims_groups,
			connector_id, connector_data, expiry
		from auth_request where id = $1;
	`, id).Scan(
		&a.ID, &a.ClientID, decoder(&a.ResponseTypes), decoder(&a.Scopes), &a.RedirectURI, &a.Nonce, &a.State,
		&a.ForceApprovalPrompt, &a.LoggedIn,
		&a.Claims.UserID, &a.Claims.Username, &a.Claims.Email, &a.Claims.EmailVerified,
		decoder(&a.Claims.Groups),
		&a.ConnectorID, &a.ConnectorData, &a.Expiry,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return a, storage.Error{Code: storage.ErrNotFound}
		}
		return a, storage.Error{Code: storage.ErrStorageMisconfigured, Details: fmt.Sprintf("select auth request: %v", err)}
	}
	return a, nil
}

func (c *conn) CreateAuthCode(a storage.AuthCode) error {
	_, err := c.Exec(`
		insert into auth_code (
			id, client_id, scopes, nonce, redirect_uri,
			claims_user_id, claims_username,
			claims_email, claims_email_verified, claims_groups,
			connector_id, connector_data,
			expiry
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);
	`,
		a.ID, a.ClientID, encoder(a.Scopes), a.Nonce, a.RedirectURI, a.Claims.UserID,
		a.Claims.Username, a.Claims.Email, a.Claims.EmailVerified, encoder(a.Claims.Groups),
		a.ConnectorID, a.ConnectorData, a.Expiry,
	)

	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.Error{Code: storage.ErrAlreadyExists}
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("insert auth code: %v", err)}
	}
	return nil
}

func (c *conn) GetAuthCode(id string) (a storage.AuthCode, err error) {
	err = c.QueryRow(`
		select
			id, client_id, scopes, nonce, redirect_uri,
			claims_user_id, claims_username,
			claims_email, claims_email_verified, claims_groups,
			connector_id, connector_data,
			expiry
		from auth_code where id = $1;
	`, id).Scan(
		&a.ID, &a.ClientID, decoder(&a.Scopes), &a.Nonce, &a.RedirectURI, &a.Claims.UserID,
		&a.Claims.Username, &a.Claims.Email, &a.Claims.EmailVerified, decoder(&a.Claims.Groups),
		&a.ConnectorID, &a.ConnectorData, &a.Expiry,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return a, storage.Error{Code: storage.ErrNotFound}
		}
		return a, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("select auth code: %v", err)}
	}
	return a, nil
}

func (c *conn) CreateRefresh(r storage.RefreshToken) error {
	_, err := c.Exec(`
		insert into refresh_token (
			id, client_id, scopes, nonce,
			claims_user_id, claims_username, claims_email, claims_email_verified,
			claims_groups,
			connector_id, connector_data,
			token, created_at, last_used
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);
	`,
		r.ID, r.ClientID, encoder(r.Scopes), r.Nonce,
		r.Claims.UserID, r.Claims.Username, r.Claims.Email, r.Claims.EmailVerified,
		encoder(r.Claims.Groups),
		r.ConnectorID, r.ConnectorData,
		r.Token, r.CreatedAt, r.LastUsed,
	)
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.Error{Code: storage.ErrAlreadyExists}
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("insert refresh_token: %v", err)}
	}
	return nil
}

func (c *conn) UpdateRefreshToken(id string, updater func(old storage.RefreshToken) (storage.RefreshToken, error)) error {
	return c.ExecTx(func(tx *trans) error {
		r, err := getRefresh(tx, id)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return err
			}
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		if r, err = updater(r); err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return err
			}
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		_, err = tx.Exec(`
			update refresh_token
			set
				client_id = $1,
				scopes = $2,
				nonce = $3,
				claims_user_id = $4,
				claims_username = $5,
				claims_email = $6,
				claims_email_verified = $7,
				claims_groups = $8,
				connector_id = $9,
				connector_data = $10,
				token = $11,
				created_at = $12,
				last_used = $13
			where
				id = $14
		`,
			r.ClientID, encoder(r.Scopes), r.Nonce,
			r.Claims.UserID, r.Claims.Username, r.Claims.Email, r.Claims.EmailVerified,
			encoder(r.Claims.Groups),
			r.ConnectorID, r.ConnectorData,
			r.Token, r.CreatedAt, r.LastUsed, id,
		)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return err
			}
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("update refresh token: %v", err)}
		}
		return nil
	})
}

func (c *conn) GetRefresh(id string) (storage.RefreshToken, error) {
	token, err := getRefresh(c, id)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return token, err
		}
		// Nope, let's make a generic storage.Error of it
		return token, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return token, nil
}

func getRefresh(q querier, id string) (storage.RefreshToken, error) {
	return scanRefresh(q.QueryRow(`
		select
			id, client_id, scopes, nonce,
			claims_user_id, claims_username, claims_email, claims_email_verified,
			claims_groups,
			connector_id, connector_data,
			token, created_at, last_used
		from refresh_token where id = $1;
	`, id))
}

func (c *conn) ListRefreshTokens() ([]storage.RefreshToken, error) {
	rows, err := c.Query(`
		select
			id, client_id, scopes, nonce,
			claims_user_id, claims_username, claims_email, claims_email_verified,
			claims_groups,
			connector_id, connector_data,
			token, created_at, last_used
		from refresh_token;
	`)
	if err != nil {
		return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("query: %v", err)}
	}
	var tokens []storage.RefreshToken
	for rows.Next() {
		r, err := scanRefresh(rows)
		if err != nil {
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		tokens = append(tokens, r)
	}
	if err := rows.Err(); err != nil {
		return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("scan: %v", err)}
	}
	return tokens, nil
}

func scanRefresh(s scanner) (r storage.RefreshToken, err error) {
	err = s.Scan(
		&r.ID, &r.ClientID, decoder(&r.Scopes), &r.Nonce,
		&r.Claims.UserID, &r.Claims.Username, &r.Claims.Email, &r.Claims.EmailVerified,
		decoder(&r.Claims.Groups),
		&r.ConnectorID, &r.ConnectorData,
		&r.Token, &r.CreatedAt, &r.LastUsed,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return r, storage.Error{Code: storage.ErrNotFound}
		}
		return r, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("scan refresh_token: %v", err)}
	}
	return r, nil
}

func (c *conn) UpdateKeys(updater func(old storage.Keys) (storage.Keys, error)) error {
	return c.ExecTx(func(tx *trans) error {
		firstUpdate := false
		// TODO(ericchiang): errors may cause a transaction be rolled back by the SQL
		// server. Test this, and consider adding a COUNT() command beforehand.
		old, err := getKeys(tx)
		if err != nil {
			if !storage.IsErrorCode(err, storage.ErrNotFound) {
				return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("get keys: %v", err)}
			}
			firstUpdate = true
			old = storage.Keys{}
		}

		nk, err := updater(old)
		if err != nil {
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
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
				return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("insert: %v", err)}
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
				return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("update: %v", err)}
			}
		}
		return nil
	})
}

func (c *conn) GetKeys() (keys storage.Keys, err error) {
	return getKeys(c)
}

func getKeys(q querier) (keys storage.Keys, err error) {
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
			return keys, storage.Error{Code: storage.ErrNotFound}
		}
		return keys, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("query keys: %v", err)}
	}
	return keys, nil
}

func (c *conn) UpdateClient(id string, updater func(old storage.Client) (storage.Client, error)) error {
	return c.ExecTx(func(tx *trans) error {
		cli, err := getClient(tx, id)
		if err != nil {
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		nc, err := updater(cli)
		if err != nil {
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
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
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("update client: %v", err)}
		}
		return nil
	})
}

func (c *conn) CreateClient(cli storage.Client) error {
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
			return storage.Error{Code: storage.ErrAlreadyExists}
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("insert client: %v", err)}
	}
	return nil
}

func getClient(q querier, id string) (storage.Client, error) {
	return scanClient(q.QueryRow(`
		select
			id, secret, redirect_uris, trusted_peers, public, name, logo_url
	    from client where id = $1;
	`, id))
}

func (c *conn) GetClient(id string) (storage.Client, error) {
	client, err := getClient(c, id)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return client, err
		}
		// Nope, let's make a generic storage.Error of it
		return client, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return client, nil
}

func (c *conn) ListClients() ([]storage.Client, error) {
	rows, err := c.Query(`
		select
			id, secret, redirect_uris, trusted_peers, public, name, logo_url
		from client;
	`)
	if err != nil {
		return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	var clients []storage.Client
	for rows.Next() {
		cli, err := scanClient(rows)
		if err != nil {
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		clients = append(clients, cli)
	}
	if err := rows.Err(); err != nil {
		return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
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
			return cli, storage.Error{Code: storage.ErrNotFound}
		}
		return cli, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("get client: %v", err)}
	}
	return cli, nil
}

func (c *conn) CreatePassword(p storage.Password) error {
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
			return storage.Error{Code: storage.ErrAlreadyExists}
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("insert password: %v", err)}
	}
	return nil
}

func (c *conn) UpdatePassword(email string, updater func(p storage.Password) (storage.Password, error)) error {
	return c.ExecTx(func(tx *trans) error {
		p, err := getPassword(tx, email)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return err
			}
			// Nope, let's make a generic storage.Error of it
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}

		np, err := updater(p)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return err
			}
			// Nope, let's make a generic storage.Error of it
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
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
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("update password: %v", err)}
		}
		return nil
	})
}

func (c *conn) GetPassword(email string) (storage.Password, error) {
	password, err := getPassword(c, email)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return password, err
		}
		// Nope, let's make a generic storage.Error of it
		return password, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return password, nil
}

func getPassword(q querier, email string) (p storage.Password, err error) {
	return scanPassword(q.QueryRow(`
		select
			email, hash, username, user_id
		from password where email = $1;
	`, strings.ToLower(email)))
}

func (c *conn) ListPasswords() ([]storage.Password, error) {
	rows, err := c.Query(`
		select
			email, hash, username, user_id
		from password;
	`)
	if err != nil {
		return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}

	var passwords []storage.Password
	for rows.Next() {
		p, err := scanPassword(rows)
		if err != nil {
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		passwords = append(passwords, p)
	}
	if err := rows.Err(); err != nil {
		return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return passwords, nil
}

func scanPassword(s scanner) (p storage.Password, err error) {
	err = s.Scan(
		&p.Email, &p.Hash, &p.Username, &p.UserID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return p, storage.Error{Code: storage.ErrNotFound}
		}
		return p, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("select password: %v", err)}
	}
	return p, nil
}

func (c *conn) CreateOfflineSessions(s storage.OfflineSessions) error {
	_, err := c.Exec(`
		insert into offline_session (
			user_id, conn_id, refresh
		)
		values (
			$1, $2, $3
		);
	`,
		s.UserID, s.ConnID, encoder(s.Refresh),
	)
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.Error{Code: storage.ErrAlreadyExists}
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("insert offline session: %v", err)}
	}
	return nil
}

func (c *conn) UpdateOfflineSessions(userID string, connID string, updater func(s storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	return c.ExecTx(func(tx *trans) error {
		s, err := getOfflineSessions(tx, userID, connID)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return err
			}
			// Nope, let's make a generic storage.Error of it
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}

		newSession, err := updater(s)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return err
			}
			// Nope, let's make a generic storage.Error of it
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		_, err = tx.Exec(`
			update offline_session
			set
				refresh = $1
			where user_id = $2 AND conn_id = $3;
		`,
			encoder(newSession.Refresh), s.UserID, s.ConnID,
		)
		if err != nil {
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("update offline session: %v", err)}
		}
		return nil
	})
}

func (c *conn) GetOfflineSessions(userID string, connID string) (storage.OfflineSessions, error) {
	session, err := getOfflineSessions(c, userID, connID)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return session, err
		}
		// Nope, let's make a generic storage.Error of it
		return session, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return session, nil
}

func getOfflineSessions(q querier, userID string, connID string) (storage.OfflineSessions, error) {
	return scanOfflineSessions(q.QueryRow(`
		select
			user_id, conn_id, refresh
		from offline_session
		where user_id = $1 AND conn_id = $2;
		`, userID, connID))
}

func scanOfflineSessions(s scanner) (o storage.OfflineSessions, err error) {
	err = s.Scan(
		&o.UserID, &o.ConnID, decoder(&o.Refresh),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return o, storage.Error{Code: storage.ErrNotFound}
		}
		return o, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("select offline session: %v", err)}
	}
	return o, nil
}

func (c *conn) CreateConnector(connector storage.Connector) error {
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
			return storage.Error{Code: storage.ErrAlreadyExists}
		}
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("insert connector: %v", err)}
	}
	return nil
}

func (c *conn) UpdateConnector(id string, updater func(s storage.Connector) (storage.Connector, error)) error {
	return c.ExecTx(func(tx *trans) error {
		connector, err := getConnector(tx, id)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return err
			}
			// Nope, let's make a generic storage.Error of it
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}

		newConn, err := updater(connector)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return err
			}
			// Nope, let's make a generic storage.Error of it
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
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
			return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("update connector: %v", err)}
		}
		return nil
	})
}

func (c *conn) GetConnector(id string) (storage.Connector, error) {
	connector, err := getConnector(c, id)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return connector, err
		}
		// Nope, let's make a generic storage.Error of it
		return connector, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return connector, nil
}

func getConnector(q querier, id string) (storage.Connector, error) {
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
			return c, storage.Error{Code: storage.ErrNotFound}
		}
		return c, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("select connector: %v", err)}
	}
	return c, nil
}

func (c *conn) ListConnectors() ([]storage.Connector, error) {
	rows, err := c.Query(`
		select
			id, type, name, resource_version, config
		from connector;
	`)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return nil, err
		}
		// Nope, let's make a generic storage.Error of it
		return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	var connectors []storage.Connector
	for rows.Next() {
		conn, err := scanConnector(rows)
		if err != nil {
			// Let's see if it is a storage.Error, if so just pass it on
			if _, ok := err.(storage.Error); ok {
				return nil, err
			}
			// Nope, let's make a generic storage.Error of it
			return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
		}
		connectors = append(connectors, conn)
	}
	if err := rows.Err(); err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return nil, err
		}
		// Nope, let's make a generic storage.Error of it
		return nil, storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return connectors, nil
}

func (c *conn) DeleteAuthRequest(id string) error {
	err := c.delete("auth_request", "id", id)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		// Nope, let's make a generic storage.Error of it
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) DeleteAuthCode(id string) error {
	err := c.delete("auth_code", "id", id)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		// Nope, let's make a generic storage.Error of it
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) DeleteClient(id string) error {
	err := c.delete("client", "id", id)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		// Nope, let's make a generic storage.Error of it
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) DeleteRefresh(id string) error {
	err := c.delete("refresh_token", "id", id)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		// Nope, let's make a generic storage.Error of it
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) DeletePassword(email string) error {
	err := c.delete("password", "email", strings.ToLower(email))
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		// Nope, let's make a generic storage.Error of it
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) DeleteConnector(id string) error {
	err := c.delete("connector", "id", id)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		// Nope, let's make a generic storage.Error of it
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: err.Error()}
	}
	return nil
}

func (c *conn) DeleteOfflineSessions(userID string, connID string) error {
	result, err := c.Exec(`delete from offline_session where user_id = $1 AND conn_id = $2`, userID, connID)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		// Nope, let's make a generic storage.Error of it
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("delete offline_session: user_id = %s, conn_id = %s", userID, connID)}
	}

	// For now mandate that the driver implements RowsAffected. If we ever need to support
	// a driver that doesn't implement this, we can run this in a transaction with a get beforehand.
	n, err := result.RowsAffected()
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		// Nope, let's make a generic storage.Error of it
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("rows affected: %v", err)}
	}
	if n < 1 {
		return storage.Error{Code: storage.ErrNotFound}
	}
	return nil
}

// Do NOT call directly. Does not escape table.
func (c *conn) delete(table, field, id string) error {
	result, err := c.Exec(`delete from `+table+` where `+field+` = $1`, id)
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		// Nope, let's make a generic storage.Error of it
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("delete %s: %v", table, id)}
	}

	// For now mandate that the driver implements RowsAffected. If we ever need to support
	// a driver that doesn't implement this, we can run this in a transaction with a get beforehand.
	n, err := result.RowsAffected()
	if err != nil {
		// Let's see if it is a storage.Error, if so just pass it on
		if _, ok := err.(storage.Error); ok {
			return err
		}
		// Nope, let's make a generic storage.Error of it
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("rows affected: %v", err)}
	}
	if n < 1 {
		return storage.Error{Code: storage.ErrNotFound}
	}
	return nil
}
