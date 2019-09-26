package sql

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dexidp/dex/storage"
)

// keysRowID is the ID of the only row we expect to populate the "keys" table.
const keysRowID = "keys"

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
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// Abstract row vs rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

type sqlValueMap map[string]driver.Value
type sqlFieldMap map[string]interface{}

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

func insert(c querier, tableName string, columns sqlValueMap) error {
	var builder strings.Builder

	builder.WriteString("INSERT INTO ")
	builder.WriteString(tableName)
	builder.WriteString(" (")

	execArguments := make([]interface{}, len(columns))

	execArgumentsPos := 0
	for columnName, columnValue := range columns {
		builder.WriteString(columnName)

		if execArgumentsPos < len(columns)-1 {
			builder.WriteString(", ")
		}

		execArguments[execArgumentsPos] = columnValue
		execArgumentsPos++
	}

	builder.WriteString(") VALUES (")

	for i := 0; i < len(columns); i++ {
		builder.WriteString("$")
		builder.WriteString(strconv.Itoa(i + 1))
		if i < len(columns)-1 {
			builder.WriteString(", ")
		}
	}

	builder.WriteString(")")

	_, err := c.Exec(builder.String(), execArguments...)

	return err
}
func update(c querier, tableName string, toUpdate sqlValueMap, where sqlValueMap) error {
	var builder strings.Builder

	builder.WriteString("UPDATE ")
	builder.WriteString(tableName)
	builder.WriteString(" SET ")

	paramList := make([]interface{}, len(toUpdate)+len(where))

	paramListPos := 0

	i := 1
	for key, value := range toUpdate {
		builder.WriteString(key)
		builder.WriteString(" = $")
		builder.WriteString(strconv.Itoa(paramListPos + 1))

		if i < len(toUpdate) {
			builder.WriteString(", ")
		}

		paramList[paramListPos] = value
		paramListPos++
		i++
	}

	builder.WriteString(" WHERE ")

	i = 1
	for key, value := range where {
		builder.WriteString(key)
		builder.WriteString(" = $")
		builder.WriteString(strconv.Itoa(paramListPos + 1))

		if i < len(where) {
			builder.WriteString(" AND ")
		}

		paramList[paramListPos] = value
		paramListPos++
		i++
	}

	_, err := c.Exec(builder.String(), paramList...)

	return err
}

func sselect(tableName string, fields []string, where sqlValueMap) (string, []interface{}) {
	var builder strings.Builder

	builder.WriteString("SELECT ")

	for i, val := range fields {
		builder.WriteString(val)

		if i < len(fields)-1 {
			builder.WriteString(", ")
		}
	}

	builder.WriteString(" FROM ")
	builder.WriteString(tableName)

	paramList := make([]interface{}, len(where))

	paramListPos := 0
	if len(where) > 0 {
		builder.WriteString(" WHERE ")
		for key, value := range where {
			builder.WriteString(key)
			builder.WriteString(" = $")
			builder.WriteString(strconv.Itoa(paramListPos + 1))

			if paramListPos < len(where)-1 {
				builder.WriteString(" AND ")
			}

			paramList[paramListPos] = value
			paramListPos++
		}
	}

	return builder.String(), paramList
}

func query(c querier, tableName string, fields []string) (*sql.Rows, error) {
	qsql, _ := sselect(tableName, fields, sqlValueMap{})
	return c.Query(qsql)
}

func queryRow(c querier, tableName string, fields []string, where sqlValueMap) scanner {
	qsql, params := sselect(tableName, fields, where)
	return c.QueryRow(qsql, params...)
}

func scanRow(c querier, tableName string, fieldMap sqlFieldMap, where sqlValueMap) error {
	fields := make([]string, len(fieldMap))
	outFields := make([]interface{}, len(fieldMap))

	i := 0
	for fieldName, fieldPtr := range fieldMap {
		fields[i] = fieldName
		outFields[i] = fieldPtr
		i++
	}

	scan := queryRow(c, tableName, fields, where)
	return scan.Scan(outFields...)
}

func (c *conn) CreateAuthRequest(a storage.AuthRequest) error {
	err := insert(c, "auth_request", sqlValueMap{
		"id":                    a.ID,
		"client_id":             a.ClientID,
		"response_types":        encoder(a.ResponseTypes),
		"scopes":                encoder(a.Scopes),
		"redirect_uri":          a.RedirectURI,
		"nonce":                 a.Nonce,
		"state":                 a.State,
		"force_approval_prompt": a.ForceApprovalPrompt,
		"logged_in":             a.LoggedIn,
		"claims_user_id":        a.Claims.UserID,
		"claims_username":       a.Claims.Username,
		"claims_email":          a.Claims.Email,
		"claims_email_verified": a.Claims.EmailVerified,
		"claims_groups":         encoder(a.Claims.Groups),
		"connector_id":          a.ConnectorID,
		"connector_data":        a.ConnectorData,
		"expiry":                a.Expiry,
	})
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert auth request: %v", err)
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
		err = update(tx, "auth_request", sqlValueMap{
			"client_id":             a.ClientID,
			"response_types":        encoder(a.ResponseTypes),
			"scopes":                encoder(a.Scopes),
			"redirect_uri":          a.RedirectURI,
			"nonce":                 a.Nonce,
			"state":                 a.State,
			"force_approval_prompt": a.ForceApprovalPrompt,
			"logged_in":             a.LoggedIn,
			"claims_user_id":        a.Claims.UserID,
			"claims_username":       a.Claims.Username,
			"claims_email":          a.Claims.Email,
			"claims_email_verified": a.Claims.EmailVerified,
			"claims_groups":         encoder(a.Claims.Groups),
			"connector_id":          a.ConnectorID,
			"connector_data":        a.ConnectorData,
			"expiry":                a.Expiry,
		}, sqlValueMap{
			"id": r.ID,
		})

		if err != nil {
			return fmt.Errorf("update auth request: %v", err)
		}
		return nil
	})

}

func (c *conn) GetAuthRequest(id string) (storage.AuthRequest, error) {
	return getAuthRequest(c, id)
}

func getAuthRequest(q querier, id string) (a storage.AuthRequest, err error) {
	err = scanRow(q, "auth_request", sqlFieldMap{
		"id":                    &a.ID,
		"client_id":             &a.ClientID,
		"response_types":        decoder(&a.ResponseTypes),
		"scopes":                decoder(&a.Scopes),
		"redirect_uri":          &a.RedirectURI,
		"nonce":                 &a.Nonce,
		"state":                 &a.State,
		"force_approval_prompt": &a.ForceApprovalPrompt,
		"logged_in":             &a.LoggedIn,
		"claims_user_id":        &a.Claims.UserID,
		"claims_username":       &a.Claims.Username,
		"claims_email":          &a.Claims.Email,
		"claims_email_verified": &a.Claims.EmailVerified,
		"claims_groups":         decoder(&a.Claims.Groups),
		"connector_id":          &a.ConnectorID,
		"connector_data":        &a.ConnectorData,
		"expiry":                &a.Expiry,
	}, sqlValueMap{
		"id": id,
	})

	if err != nil {
		if err == sql.ErrNoRows {
			return a, storage.ErrNotFound
		}
		return a, fmt.Errorf("select auth request: %v", err)
	}
	return a, nil
}

func (c *conn) CreateAuthCode(a storage.AuthCode) error {
	err := insert(c, "auth_code", sqlValueMap{
		"id":                    a.ID,
		"client_id":             a.ClientID,
		"scopes":                encoder(a.Scopes),
		"nonce":                 a.Nonce,
		"redirect_uri":          a.RedirectURI,
		"claims_user_id":        a.Claims.UserID,
		"claims_username":       a.Claims.Username,
		"claims_email":          a.Claims.Email,
		"claims_email_verified": a.Claims.EmailVerified,
		"claims_groups":         encoder(a.Claims.Groups),
		"connector_id":          a.ConnectorID,
		"connector_data":        a.ConnectorData,
		"expiry":                a.Expiry,
	})

	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert auth code: %v", err)
	}
	return nil
}

func (c *conn) GetAuthCode(id string) (a storage.AuthCode, err error) {
	err = scanRow(c, "auth_code", sqlFieldMap{
		"id":                    &a.ID,
		"client_id":             &a.ClientID,
		"scopes":                decoder(&a.Scopes),
		"nonce":                 &a.Nonce,
		"redirect_uri":          &a.RedirectURI,
		"claims_user_id":        &a.Claims.UserID,
		"claims_username":       &a.Claims.Username,
		"claims_email":          &a.Claims.Email,
		"claims_email_verified": &a.Claims.EmailVerified,
		"claims_groups":         decoder(&a.Claims.Groups),
		"connector_id":          &a.ConnectorID,
		"connector_data":        &a.ConnectorData,
		"expiry":                &a.Expiry,
	}, sqlValueMap{
		"id": id,
	})

	if err != nil {
		if err == sql.ErrNoRows {
			return a, storage.ErrNotFound
		}
		return a, fmt.Errorf("select auth code: %v", err)
	}
	return a, nil
}

func (c *conn) CreateRefresh(r storage.RefreshToken) error {
	err := insert(c, "refresh_token", sqlValueMap{
		"id":                    r.ID,
		"client_id":             r.ClientID,
		"scopes":                encoder(r.Scopes),
		"nonce":                 r.Nonce,
		"claims_user_id":        r.Claims.UserID,
		"claims_username":       r.Claims.Username,
		"claims_email":          r.Claims.Email,
		"claims_email_verified": r.Claims.EmailVerified,
		"claims_groups":         encoder(r.Claims.Groups),
		"connector_id":          r.ConnectorID,
		"connector_data":        r.ConnectorData,
		"token":                 r.Token,
		"created_at":            r.CreatedAt,
		"last_used":             r.LastUsed,
	})

	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert refresh_token: %v", err)
	}
	return nil
}

func (c *conn) UpdateRefreshToken(id string, updater func(old storage.RefreshToken) (storage.RefreshToken, error)) error {
	return c.ExecTx(func(tx *trans) error {
		r, err := getRefresh(tx, id)
		if err != nil {
			return err
		}
		if r, err = updater(r); err != nil {
			return err
		}
		err = update(tx, "refresh_token", sqlValueMap{
			"client_id":             r.ClientID,
			"scopes":                encoder(r.Scopes),
			"nonce":                 r.Nonce,
			"claims_user_id":        r.Claims.UserID,
			"claims_username":       r.Claims.Username,
			"claims_email":          r.Claims.Email,
			"claims_email_verified": r.Claims.EmailVerified,
			"claims_groups":         encoder(r.Claims.Groups),
			"connector_id":          r.ConnectorID,
			"connector_data":        r.ConnectorData,
			"token":                 r.Token,
			"created_at":            r.CreatedAt,
			"last_used":             r.LastUsed,
		}, sqlValueMap{
			"id": id,
		})

		if err != nil {
			return fmt.Errorf("update refresh token: %v", err)
		}
		return nil
	})
}

func (c *conn) GetRefresh(id string) (storage.RefreshToken, error) {
	return getRefresh(c, id)
}

var refreshTokenFieldList = []string{
	"id",
	"client_id",
	"scopes",
	"nonce",
	"claims_user_id",
	"claims_username",
	"claims_email",
	"claims_email_verified",
	"claims_groups",
	"connector_id",
	"connector_data",
	"token",
	"created_at",
	"last_used",
}

func getRefresh(q querier, id string) (storage.RefreshToken, error) {
	return scanRefresh(queryRow(q, "refresh_token", refreshTokenFieldList, sqlValueMap{
		"id": id,
	}))
}

func (c *conn) ListRefreshTokens() ([]storage.RefreshToken, error) {
	rows, err := query(c, "refresh_token", refreshTokenFieldList)

	if err != nil {
		return nil, fmt.Errorf("query: %v", err)
	}
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
		&r.Claims.UserID, &r.Claims.Username, &r.Claims.Email, &r.Claims.EmailVerified,
		decoder(&r.Claims.Groups),
		&r.ConnectorID, &r.ConnectorData,
		&r.Token, &r.CreatedAt, &r.LastUsed,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return r, storage.ErrNotFound
		}
		return r, fmt.Errorf("scan refresh_token: %v", err)
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
			err := insert(tx, "keys", sqlValueMap{
				"id":                keysRowID,
				"verification_keys": encoder(nk.VerificationKeys),
				"signing_key":       encoder(nk.SigningKey),
				"signing_key_pub":   encoder(nk.SigningKeyPub),
				"next_rotation":     nk.NextRotation,
			})

			if err != nil {
				return fmt.Errorf("insert: %v", err)
			}
		} else {
			err = update(tx, "keys", sqlValueMap{
				"verification_keys": encoder(nk.VerificationKeys),
				"signing_key":       encoder(nk.SigningKey),
				"signing_key_pub":   encoder(nk.SigningKeyPub),
				"next_rotation":     nk.NextRotation,
			}, sqlValueMap{
				"id": keysRowID,
			})
			if err != nil {
				return fmt.Errorf("update: %v", err)
			}
		}
		return nil
	})
}

func (c *conn) GetKeys() (keys storage.Keys, err error) {
	return getKeys(c)
}

func getKeys(q querier) (keys storage.Keys, err error) {
	err = scanRow(q, "keys", sqlFieldMap{
		"verification_keys": decoder(&keys.VerificationKeys),
		"signing_key":       decoder(&keys.SigningKey),
		"signing_key_pub":   decoder(&keys.SigningKeyPub),
		"next_rotation":     &keys.NextRotation,
	}, sqlValueMap{
		"id": keysRowID,
	})

	if err != nil {
		if err == sql.ErrNoRows {
			return keys, storage.ErrNotFound
		}
		return keys, fmt.Errorf("query keys: %v", err)
	}
	return keys, nil
}

func (c *conn) UpdateClient(id string, updater func(old storage.Client) (storage.Client, error)) error {
	return c.ExecTx(func(tx *trans) error {
		cli, err := getClient(tx, id)
		if err != nil {
			return err
		}
		nc, err := updater(cli)
		if err != nil {
			return err
		}

		err = update(tx, "client", sqlValueMap{
			"secret":        nc.Secret,
			"redirect_uris": encoder(nc.RedirectURIs),
			"trusted_peers": encoder(nc.TrustedPeers),
			"public":        nc.Public,
			"name":          nc.Name,
			"logo_url":      nc.LogoURL,
		}, sqlValueMap{
			"id": id,
		})
		if err != nil {
			return fmt.Errorf("update client: %v", err)
		}
		return nil
	})
}

func (c *conn) CreateClient(cli storage.Client) error {
	err := insert(c, "client", sqlValueMap{
		"id":            cli.ID,
		"secret":        cli.Secret,
		"redirect_uris": encoder(cli.RedirectURIs),
		"trusted_peers": encoder(cli.TrustedPeers),
		"public":        cli.Public,
		"name":          cli.Name,
		"logo_url":      cli.LogoURL,
	})

	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert client: %v", err)
	}
	return nil
}

var clientFieldList = []string{
	"id",
	"secret",
	"redirect_uris",
	"trusted_peers",
	"public",
	"name",
	"logo_url",
}

func getClient(q querier, id string) (storage.Client, error) {
	return scanClient(queryRow(q, "client", clientFieldList, sqlValueMap{
		"id": id,
	}))
}

func (c *conn) GetClient(id string) (storage.Client, error) {
	return getClient(c, id)
}

func (c *conn) ListClients() ([]storage.Client, error) {
	rows, err := query(c, "client", clientFieldList)
	if err != nil {
		return nil, err
	}
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

func (c *conn) CreatePassword(p storage.Password) error {
	p.Email = strings.ToLower(p.Email)
	err := insert(c, "password", sqlValueMap{
		"email":    p.Email,
		"hash":     p.Hash,
		"username": p.Username,
		"user_id":  p.UserID,
	})

	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert password: %v", err)
	}
	return nil
}

func (c *conn) UpdatePassword(email string, updater func(p storage.Password) (storage.Password, error)) error {
	return c.ExecTx(func(tx *trans) error {
		p, err := getPassword(tx, email)
		if err != nil {
			return err
		}

		np, err := updater(p)
		if err != nil {
			return err
		}
		err = update(tx, "password", sqlValueMap{
			"hash":     np.Hash,
			"username": np.Username,
			"user_id":  np.UserID,
		}, sqlValueMap{
			"email": p.Email,
		})
		if err != nil {
			return fmt.Errorf("update password: %v", err)
		}
		return nil
	})
}

func (c *conn) GetPassword(email string) (storage.Password, error) {
	return getPassword(c, email)
}

var passwordFieldList = []string{
	"email",
	"hash",
	"username",
	"user_id",
}

func getPassword(q querier, email string) (p storage.Password, err error) {
	return scanPassword(queryRow(q, "password", passwordFieldList, sqlValueMap{
		"email": strings.ToLower(email),
	}))
}

func (c *conn) ListPasswords() ([]storage.Password, error) {
	rows, err := query(c, "password", passwordFieldList)
	if err != nil {
		return nil, err
	}

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

func (c *conn) CreateOfflineSessions(s storage.OfflineSessions) error {
	err := insert(c, "offline_session", sqlValueMap{
		"user_id": s.UserID,
		"conn_id": s.ConnID,
		"refresh": encoder(s.Refresh),
	})
	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert offline session: %v", err)
	}
	return nil
}

func (c *conn) UpdateOfflineSessions(userID string, connID string, updater func(s storage.OfflineSessions) (storage.OfflineSessions, error)) error {
	return c.ExecTx(func(tx *trans) error {
		s, err := getOfflineSessions(tx, userID, connID)
		if err != nil {
			return err
		}

		newSession, err := updater(s)
		if err != nil {
			return err
		}

		err = update(tx, "offline_session", sqlValueMap{
			"refresh": encoder(newSession.Refresh),
		}, sqlValueMap{
			"user_id": s.UserID,
			"conn_id": s.ConnID,
		})
		if err != nil {
			return fmt.Errorf("update offline session: %v", err)
		}
		return nil
	})
}

func (c *conn) GetOfflineSessions(userID string, connID string) (storage.OfflineSessions, error) {
	return getOfflineSessions(c, userID, connID)
}

func getOfflineSessions(q querier, userID string, connID string) (o storage.OfflineSessions, err error) {
	err = scanRow(q, "offline_session", sqlFieldMap{
		"user_id": &o.UserID,
		"conn_id": &o.ConnID,
		"refresh": decoder(&o.Refresh),
	}, sqlValueMap{
		"user_id": userID,
		"conn_id": connID,
	})

	if err != nil {
		if err == sql.ErrNoRows {
			return o, storage.ErrNotFound
		}
		return o, fmt.Errorf("select offline session: %v", err)
	}
	return
}

func (c *conn) CreateConnector(connector storage.Connector) error {
	err := insert(c, "connector", sqlValueMap{
		"id":               connector.ID,
		"type":             connector.Type,
		"name":             connector.Name,
		"resource_version": connector.ResourceVersion,
		"config":           connector.Config,
	})

	if err != nil {
		if c.alreadyExistsCheck(err) {
			return storage.ErrAlreadyExists
		}
		return fmt.Errorf("insert connector: %v", err)
	}
	return nil
}

func (c *conn) UpdateConnector(id string, updater func(s storage.Connector) (storage.Connector, error)) error {
	return c.ExecTx(func(tx *trans) error {
		connector, err := getConnector(tx, id)
		if err != nil {
			return err
		}

		newConn, err := updater(connector)
		if err != nil {
			return err
		}
		err = update(tx, "connector", sqlValueMap{
			"type":             newConn.Type,
			"name":             newConn.Name,
			"resource_version": newConn.ResourceVersion,
			"config":           newConn.Config,
		}, sqlValueMap{
			"id": connector.ID,
		})
		if err != nil {
			return fmt.Errorf("update connector: %v", err)
		}
		return nil
	})
}

func (c *conn) GetConnector(id string) (storage.Connector, error) {
	return getConnector(c, id)
}

var connectorFieldList = []string{
	"id",
	"type",
	"name",
	"resource_version",
	"config",
}

func getConnector(q querier, id string) (storage.Connector, error) {
	return scanConnector(queryRow(q, "connector", connectorFieldList, sqlValueMap{
		"id": id,
	}))
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

func (c *conn) ListConnectors() ([]storage.Connector, error) {
	rows, err := query(c, "connector", connectorFieldList)
	if err != nil {
		return nil, err
	}
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

func (c *conn) DeleteAuthRequest(id string) error { return c.delete("auth_request", "id", id) }
func (c *conn) DeleteAuthCode(id string) error    { return c.delete("auth_code", "id", id) }
func (c *conn) DeleteClient(id string) error      { return c.delete("client", "id", id) }
func (c *conn) DeleteRefresh(id string) error     { return c.delete("refresh_token", "id", id) }
func (c *conn) DeletePassword(email string) error {
	return c.delete("password", "email", strings.ToLower(email))
}
func (c *conn) DeleteConnector(id string) error { return c.delete("connector", "id", id) }

func (c *conn) DeleteOfflineSessions(userID string, connID string) error {
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
