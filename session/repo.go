package session

import "time"

type SessionRepo interface {
	Get(string) (*Session, error)
	Create(Session) error
	Update(Session) error
}

type SessionKeyRepo interface {
	Push(SessionKey, time.Duration) error
	Pop(string) (string, error)
	Peek(string) (string, error)
}
