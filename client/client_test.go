package client

import (
	"encoding/json"
	"net/url"
	"reflect"
	"sort"
	"testing"

	"github.com/coreos/go-oidc/oidc"
)

func TestMemClientIdentityRepoNew(t *testing.T) {
	tests := []struct {
		id   string
		meta oidc.ClientMetadata
	}{
		{
			id: "foo",
			meta: oidc.ClientMetadata{
				RedirectURIs: []url.URL{
					url.URL{
						Scheme: "https",
						Host:   "example.com",
					},
				},
			},
		},
		{
			id: "bar",
			meta: oidc.ClientMetadata{
				RedirectURIs: []url.URL{
					url.URL{Scheme: "https", Host: "example.com/foo"},
					url.URL{Scheme: "https", Host: "example.com/bar"},
				},
			},
		},
	}

	for i, tt := range tests {
		cr := NewClientIdentityRepo(nil)
		creds, err := cr.New(tt.id, tt.meta)
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
		}

		if creds.ID != tt.id {
			t.Errorf("case %d: expected non-empty Client ID", i)
		}

		if creds.Secret == "" {
			t.Errorf("case %d: expected non-empty Client Secret", i)
		}

		all, err := cr.All()
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
		}
		if len(all) != 1 {
			t.Errorf("case %d: expected repo to contain newly created Client", i)
		}

		wantURLs := tt.meta.RedirectURIs
		gotURLs := all[0].Metadata.RedirectURIs
		if !reflect.DeepEqual(wantURLs, gotURLs) {
			t.Errorf("case %d: redirect url mismatch, want=%v, got=%v", i, wantURLs, gotURLs)
		}
	}
}

func TestMemClientIdentityRepoNewDuplicate(t *testing.T) {
	cr := NewClientIdentityRepo(nil)

	meta1 := oidc.ClientMetadata{
		RedirectURIs: []url.URL{
			url.URL{Scheme: "https", Host: "foo.example.com"},
		},
	}

	if _, err := cr.New("foo", meta1); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	meta2 := oidc.ClientMetadata{
		RedirectURIs: []url.URL{
			url.URL{Scheme: "https", Host: "bar.example.com"},
		},
	}

	if _, err := cr.New("foo", meta2); err == nil {
		t.Errorf("expected non-nil error")
	}
}

func TestMemClientIdentityRepoAll(t *testing.T) {
	tests := []struct {
		ids []string
	}{
		{
			ids: nil,
		},
		{
			ids: []string{"foo"},
		},
		{
			ids: []string{"foo", "bar"},
		},
	}

	for i, tt := range tests {
		cs := make([]oidc.ClientIdentity, len(tt.ids))
		for i, s := range tt.ids {
			cs[i] = oidc.ClientIdentity{
				Credentials: oidc.ClientCredentials{
					ID: s,
				},
			}
		}

		cr := NewClientIdentityRepo(cs)

		all, err := cr.All()
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
		}

		want := sortableClientIdentities(cs)
		sort.Sort(want)
		got := sortableClientIdentities(all)
		sort.Sort(got)

		if len(got) != len(want) {
			t.Errorf("case %d: wrong length: %d", i, len(got))
		}

		if !reflect.DeepEqual(want, got) {
			t.Errorf("case %d: want=%#v, got=%#v", i, want, got)
		}
	}
}

func TestClientIdentityUnmarshalJSON(t *testing.T) {
	for i, test := range []struct {
		json           string
		expectedID     string
		expectedSecret string
		expectedURLs   []string
	}{
		{
			json:           `{"id":"12345","secret":"rosebud","redirectURLs":["https://redirectone.com", "https://redirecttwo.com"]}`,
			expectedID:     "12345",
			expectedSecret: "rosebud",
			expectedURLs: []string{
				"https://redirectone.com",
				"https://redirecttwo.com",
			},
		},
	} {
		var actual clientIdentity
		err := json.Unmarshal([]byte(test.json), &actual)
		if err != nil {
			t.Errorf("case %d: error unmarshalling: %v", i, err)
			continue
		}

		if actual.Credentials.ID != test.expectedID {
			t.Errorf("case %d: actual.Credentials.ID == %v, want %v", i, actual.Credentials.ID, test.expectedID)
		}

		if actual.Credentials.Secret != test.expectedSecret {
			t.Errorf("case %d: actual.Credentials.Secret == %v, want %v", i, actual.Credentials.Secret, test.expectedSecret)
		}
		expectedURLs := test.expectedURLs
		sort.Strings(expectedURLs)

		actualURLs := make([]string, 0)
		for _, u := range actual.Metadata.RedirectURIs {
			actualURLs = append(actualURLs, u.String())
		}
		sort.Strings(actualURLs)
		if len(actualURLs) != len(expectedURLs) {
			t.Errorf("case %d: len(actualURLs) == %v, want %v", i, len(actualURLs), len(expectedURLs))
		}
		for ui, actualURL := range actualURLs {
			if actualURL != expectedURLs[ui] {
				t.Errorf("case %d: actualURLs[%d] == %q, want %q", i, ui, actualURL, expectedURLs[ui])
			}
		}
	}
}
