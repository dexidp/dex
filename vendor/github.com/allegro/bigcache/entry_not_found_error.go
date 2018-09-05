package bigcache

import "fmt"

// EntryNotFoundError is an error type struct which is returned when entry was not found for provided key
type EntryNotFoundError struct {
	message string
}

func notFound(key string) error {
	return &EntryNotFoundError{fmt.Sprintf("Entry %q not found", key)}
}

// Error returned when entry does not exist.
func (e EntryNotFoundError) Error() string {
	return e.message
}
