package memory

import (
	"os"
	"reflect"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/dex/storage"
)

func TestStaticClients(t *testing.T) {
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}
	s := New(logger)

	c1 := storage.Client{ID: "foo", Secret: "foo_secret"}
	c2 := storage.Client{ID: "bar", Secret: "bar_secret"}
	s.CreateClient(c1)
	s2 := storage.WithStaticClients(s, []storage.Client{c2})

	tests := []struct {
		id         string
		s          storage.Storage
		wantErr    bool
		wantClient storage.Client
	}{
		{"foo", s, false, c1},
		{"bar", s, true, storage.Client{}},
		{"foo", s2, true, storage.Client{}},
		{"bar", s2, false, c2},
	}

	for i, tc := range tests {
		gotClient, err := tc.s.GetClient(tc.id)
		if err != nil {
			if !tc.wantErr {
				t.Errorf("case %d: GetClient(%q) %v", i, tc.id, err)
			}
			continue
		}

		if tc.wantErr {
			t.Errorf("case %d: GetClient(%q) expected error", i, tc.id)
			continue
		}

		if !reflect.DeepEqual(tc.wantClient, gotClient) {
			t.Errorf("case %d: expected=%#v got=%#v", i, tc.wantClient, gotClient)
		}
	}
}
