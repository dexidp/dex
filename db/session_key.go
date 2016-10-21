package db

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/jonboulle/clockwork"

	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/dex/session"
)

const (
	sessionKeyTableName = "session_key"
)

func init() {
	register(table{
		name:    sessionKeyTableName,
		model:   sessionKeyModel{},
		autoinc: false,
		pkey:    []string{"key"},
	})
}

type sessionKeyModel struct {
	Key       string `db:"key"`
	SessionID string `db:"session_id"`
	ExpiresAt int64  `db:"expires_at"`
	Stale     bool   `db:"stale"`
}

func NewSessionKeyRepo(dbm *gorp.DbMap) *SessionKeyRepo {
	return NewSessionKeyRepoWithClock(dbm, clockwork.NewRealClock())
}

func NewSessionKeyRepoWithClock(dbm *gorp.DbMap, clock clockwork.Clock) *SessionKeyRepo {
	return &SessionKeyRepo{db: &db{dbm}, clock: clock}
}

type SessionKeyRepo struct {
	*db
	clock clockwork.Clock
}

func (r *SessionKeyRepo) Push(sk session.SessionKey, exp time.Duration) error {
	skm := &sessionKeyModel{
		Key:       sk.Key,
		SessionID: sk.SessionID,
		ExpiresAt: r.clock.Now().Unix() + int64(exp.Seconds()),
		Stale:     false,
	}
	return r.executor(nil).Insert(skm)
}

func (r *SessionKeyRepo) Pop(key string) (string, error) {
	m, err := r.executor(nil).Get(sessionKeyModel{}, key)
	if err != nil {
		return "", err
	}

	if m == nil {
		return "", errors.New("session key does not exist")
	}

	skm, ok := m.(*sessionKeyModel)
	if !ok {
		log.Errorf("expected sessionKeyModel but found %v", reflect.TypeOf(m))
		return "", errors.New("unrecognized model")
	}

	if skm.Stale || skm.ExpiresAt < r.clock.Now().Unix() {
		return "", errors.New("invalid session key")
	}

	qt := r.quote(sessionKeyTableName)
	q := fmt.Sprintf("UPDATE %s SET stale=$1 WHERE key=$2 AND stale=$3", qt)
	res, err := r.executor(nil).Exec(q, true, key, false)
	if err != nil {
		return "", err
	}

	if n, err := res.RowsAffected(); n != 1 {
		if err != nil {
			log.Errorf("Failed determining rows affected by UPDATE session_key query: %v", err)
		}
		return "", fmt.Errorf("failed to pop entity")
	}

	return skm.SessionID, nil
}

func (r *SessionKeyRepo) Peek(key string) (string, error) {
	m, err := r.executor(nil).Get(sessionKeyModel{}, key)
	if err != nil {
		return "", err
	}

	if m == nil {
		return "", errors.New("session key does not exist")
	}

	skm, ok := m.(*sessionKeyModel)
	if !ok {
		log.Errorf("expected sessionKeyModel but found %v", reflect.TypeOf(m))
		return "", errors.New("unrecognized model")
	}

	if skm.Stale || skm.ExpiresAt < r.clock.Now().Unix() {
		return "", errors.New("invalid session key")
	}

	return skm.SessionID, nil
}

func (r *SessionKeyRepo) purge() error {
	qt := r.quote(sessionKeyTableName)
	q := fmt.Sprintf("DELETE FROM %s WHERE stale = $1 OR expires_at < $2", qt)
	res, err := r.executor(nil).Exec(q, true, r.clock.Now().Unix())
	if err != nil {
		return err
	}

	d := "unknown # of"
	if n, err := res.RowsAffected(); err == nil {
		if n == 0 {
			return nil
		}
		d = fmt.Sprintf("%d", n)
	}

	log.Infof("Deleted %s stale row(s) from %s table", d, sessionKeyTableName)
	return nil
}
