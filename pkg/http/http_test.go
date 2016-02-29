package http

import (
	"net/url"
	"reflect"
	"testing"
)

func TestMergeQuery(t *testing.T) {
	tests := []struct {
		u string
		q url.Values
		w string
	}{
		// No values
		{
			u: "http://example.com",
			q: nil,
			w: "http://example.com",
		},
		// No additional values
		{
			u: "http://example.com?foo=bar",
			q: nil,
			w: "http://example.com?foo=bar",
		},
		// Simple addition
		{
			u: "http://example.com",
			q: url.Values{
				"foo": []string{"bar"},
			},
			w: "http://example.com?foo=bar",
		},
		// Addition with existing values
		{
			u: "http://example.com?dog=boo",
			q: url.Values{
				"foo": []string{"bar"},
			},
			w: "http://example.com?dog=boo&foo=bar",
		},
		// Merge
		{
			u: "http://example.com?dog=boo",
			q: url.Values{
				"dog": []string{"elroy"},
			},
			w: "http://example.com?dog=boo&dog=elroy",
		},
		// Add and merge
		{
			u: "http://example.com?dog=boo",
			q: url.Values{
				"dog": []string{"elroy"},
				"foo": []string{"bar"},
			},
			w: "http://example.com?dog=boo&dog=elroy&foo=bar",
		},
		// Multivalue merge
		{
			u: "http://example.com?dog=boo",
			q: url.Values{
				"dog": []string{"elroy", "penny"},
			},
			w: "http://example.com?dog=boo&dog=elroy&dog=penny",
		},
	}

	for i, tt := range tests {
		ur, err := url.Parse(tt.u)
		if err != nil {
			t.Errorf("case %d: failed parsing test url: %v, error: %v", i, tt.u, err)
		}

		got := MergeQuery(*ur, tt.q)
		want, err := url.Parse(tt.w)
		if err != nil {
			t.Errorf("case %d: failed parsing want url: %v, error: %v", i, tt.w, err)
		}

		if !reflect.DeepEqual(*want, got) {
			t.Errorf("case %d: want: %v, got: %v", i, *want, got)
		}
	}
}

func TestNewResourceLocation(t *testing.T) {
	tests := []struct {
		ru   *url.URL
		id   string
		want string
	}{
		{
			ru: &url.URL{
				Scheme: "http",
				Host:   "example.com",
			},
			id:   "foo",
			want: "http://example.com/foo",
		},
		// https
		{
			ru: &url.URL{
				Scheme: "https",
				Host:   "example.com",
			},
			id:   "foo",
			want: "https://example.com/foo",
		},
		// with path
		{
			ru: &url.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "one/two/three",
			},
			id:   "foo",
			want: "http://example.com/one/two/three/foo",
		},
		// with fragment
		{
			ru: &url.URL{
				Scheme:   "http",
				Host:     "example.com",
				Fragment: "frag",
			},
			id:   "foo",
			want: "http://example.com/foo",
		},
		// with query
		{
			ru: &url.URL{
				Scheme:   "http",
				Host:     "example.com",
				RawQuery: "dog=elroy",
			},
			id:   "foo",
			want: "http://example.com/foo",
		},
	}

	for i, tt := range tests {
		got := NewResourceLocation(tt.ru, tt.id)
		if tt.want != got {
			t.Errorf("case %d: want=%s, got=%s", i, tt.want, got)
		}
	}
}
