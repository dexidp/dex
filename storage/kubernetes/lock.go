package kubernetes

import (
	"fmt"
	"time"
)

const (
	lockAnnotation = "dexidp.com/resource-lock"
	lockTimeFormat = time.RFC3339
)

var (
	lockTimeout     = 10 * time.Second
	lockCheckPeriod = 100 * time.Millisecond
)

// refreshTokenLock is an implementation of annotation-based optimistic locking.
//
// Refresh token contains data to refresh identity in external authentication system.
// There is a requirement that refresh should be called only once because of several reasons:
//   - Some of OIDC providers could use the refresh token rotation feature which requires calling refresh only once.
//   - Providers can limit the rate of requests to the token endpoint, which will lead to the error
//     in case of many concurrent requests.
type refreshTokenLock struct {
	cli          *client
	waitingState bool
}

func newRefreshTokenLock(cli *client) *refreshTokenLock {
	return &refreshTokenLock{cli: cli}
}

func (l *refreshTokenLock) Lock(id string) error {
	for i := 0; i <= 60; i++ {
		ok, err := l.setLockAnnotation(id)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		time.Sleep(lockCheckPeriod)
	}
	return fmt.Errorf("timeout waiting for refresh token %s lock", id)
}

func (l *refreshTokenLock) Unlock(id string) {
	if l.waitingState {
		// Do not need to unlock for waiting goroutines, because the have not set it.
		return
	}

	r, err := l.cli.getRefreshToken(id)
	if err != nil {
		l.cli.logger.Debug("failed to get resource to release lock for refresh token", "token_id", id, "err", err)
		return
	}

	r.Annotations = nil
	err = l.cli.put(resourceRefreshToken, r.Name, r)
	if err != nil {
		l.cli.logger.Debug("failed to release lock for refresh token", "token_id", id, "err", err)
	}
}

func (l *refreshTokenLock) setLockAnnotation(id string) (bool, error) {
	r, err := l.cli.getRefreshToken(id)
	if err != nil {
		return false, err
	}

	currentTime := time.Now()
	lockData := map[string]string{
		lockAnnotation: currentTime.Add(lockTimeout).Format(lockTimeFormat),
	}

	val, ok := r.Annotations[lockAnnotation]
	if !ok {
		if l.waitingState {
			return false, nil
		}

		r.Annotations = lockData
		err := l.cli.put(resourceRefreshToken, r.Name, r)
		if err == nil {
			return false, nil
		}

		if isKubernetesAPIConflictError(err) {
			l.waitingState = true
			return true, nil
		}
		return false, err
	}

	until, err := time.Parse(lockTimeFormat, val)
	if err != nil {
		return false, fmt.Errorf("lock annotation value is malformed: %v", err)
	}

	if !currentTime.After(until) {
		// waiting for the lock to be released
		l.waitingState = true
		return true, nil
	}

	// Lock time is out, lets break the lock and take the advantage
	r.Annotations = lockData

	err = l.cli.put(resourceRefreshToken, r.Name, r)
	if err == nil {
		// break lock annotation
		return false, nil
	}

	l.cli.logger.Debug("break lock annotation", "error", err)
	if isKubernetesAPIConflictError(err) {
		l.waitingState = true
		// after breaking error waiting for the lock to be released
		return true, nil
	}
	return false, err
}
