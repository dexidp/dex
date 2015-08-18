package workerschema

import (
	"net/http"
	"strings"
)

func NewWithBasePath(client *http.Client, basePath string) (*Service, error) {
	s, err := New(client)
	if err == nil {
		s.BasePath = strings.Replace(s.BasePath, "$ENDPOINT", basePath, 1)
	}
	return s, nil
}
