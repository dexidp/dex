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
//
// The lock uses a Kubernetes annotation on the refresh token resource as a mutex.
// Only one goroutine can hold the lock at a time; others poll until the annotation
// is removed (unlocked) or expires (broken). The Kubernetes resourceVersion on put
// acts as compare-and-swap: if two goroutines race to set the annotation, only one
// succeeds and the other gets a 409 Conflict.
type refreshTokenLock struct {
	cli *client
	// waitingState tracks whether this lock instance has lost a compare-and-swap race
	// and is now polling for the lock to be released. Used by Unlock to skip the
	// annotation removal — only the goroutine that successfully wrote the annotation
	// should remove it.
	waitingState bool
}

func newRefreshTokenLock(cli *client) *refreshTokenLock {
	return &refreshTokenLock{cli: cli}
}

// Lock polls until the lock annotation can be set on the refresh token resource.
// Returns nil when the lock is acquired, or an error on timeout (60 attempts × 100ms).
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

// Unlock removes the lock annotation from the refresh token resource.
// Only the holder of the lock (waitingState == false) performs the removal.
func (l *refreshTokenLock) Unlock(id string) {
	if l.waitingState {
		// This goroutine never successfully wrote the annotation, so there's
		// nothing to remove. Another goroutine holds (or held) the lock.
		return
	}

	r, err := l.cli.getRefreshToken(id)
	if err != nil {
		l.cli.logger.Debug("failed to get resource to release lock for refresh token", "token_id", id, "err", err)
		return
	}

	r.Annotations = nil
	err = l.cli.put(resourceRefreshToken, r.ObjectMeta.Name, r)
	if err != nil {
		l.cli.logger.Debug("failed to release lock for refresh token", "token_id", id, "err", err)
	}
}

// setLockAnnotation attempts to acquire the lock by writing an annotation with
// an expiration timestamp. Returns (true, nil) when the caller should keep waiting,
// (false, nil) when the lock is acquired, or (false, err) on a non-retriable error.
//
// The locking protocol relies on Kubernetes optimistic concurrency: every put
// includes the resource's current resourceVersion, so concurrent writes to the
// same object result in a 409 Conflict for all but one writer.
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
		// No annotation means the lock is free. Every goroutine — whether it's
		// a first-time caller or was previously waiting — must compete by writing
		// the annotation. The put uses the current resourceVersion, so only one
		// writer succeeds; the rest get a 409 Conflict and go back to polling.
		r.Annotations = lockData
		err := l.cli.put(resourceRefreshToken, r.ObjectMeta.Name, r)
		if err == nil {
			l.waitingState = false
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
		// Lock is held by another goroutine and has not expired yet — keep polling.
		l.waitingState = true
		return true, nil
	}

	// Lock has expired (holder crashed or is too slow). Attempt to break it by
	// overwriting the annotation with a new expiration. Again, only one writer
	// can win the compare-and-swap race.
	r.Annotations = lockData

	err = l.cli.put(resourceRefreshToken, r.ObjectMeta.Name, r)
	if err == nil {
		return false, nil
	}

	l.cli.logger.Debug("break lock annotation", "error", err)
	if isKubernetesAPIConflictError(err) {
		l.waitingState = true
		return true, nil
	}
	return false, err
}
