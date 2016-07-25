package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"sync"
	"time"

	"golang.org/x/net/context"
)

var (
	gcInterval = time.Minute
	expiresIn  = time.Minute * 10
)

type memNonceSource struct {
	mu     sync.Mutex
	nonces map[string]time.Time
}

func newNonceSource(ctx context.Context) *memNonceSource {
	s := &memNonceSource{nonces: make(map[string]time.Time)}
	go s.garbageCollect(ctx)
	return s
}

func (s *memNonceSource) Nonce() (string, error) {
	buff := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buff); err != nil {
		return "", err
	}
	nonce := base64.RawURLEncoding.EncodeToString(buff)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonces[nonce] = time.Now().Add(expiresIn)

	return nonce, nil
}

func (s *memNonceSource) ClaimNonce(nonce string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.nonces[nonce]; ok {
		delete(s.nonces, nonce)
		return nil
	}
	return errors.New("invalid nonce")
}

func (s *memNonceSource) garbageCollect(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
		case <-time.After(gcInterval):
			s.mu.Lock()
			now := time.Now()

			for nonce, exp := range s.nonces {
				if now.After(exp) {
					delete(s.nonces, nonce)
				}
			}

			s.mu.Unlock()
		}
	}
}
