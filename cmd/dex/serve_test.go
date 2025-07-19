package main

import (
	"net/http"
	"net/url"
	"testing"
)

func Test_omitStatic(t *testing.T) {
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "static prefix",
			args: args{
				r: &http.Request{URL: &url.URL{Path: "/dex/static/file.css"}},
			},
			want: false,
		},
		{
			name: "theme prefix",
			args: args{
				r: &http.Request{URL: &url.URL{Path: "/dex/theme/logo.png"}},
			},
			want: false,
		},
		{
			name: "root path",
			args: args{
				r: &http.Request{URL: &url.URL{Path: "/"}},
			},
			want: true,
		},
		{
			name: "other dex path",
			args: args{
				r: &http.Request{URL: &url.URL{Path: "/dex/other"}},
			},
			want: true,
		},
		{
			name: "non-dex path",
			args: args{
				r: &http.Request{URL: &url.URL{Path: "/api"}},
			},
			want: true,
		},
		{
			name: "case sensitive static",
			args: args{
				r: &http.Request{URL: &url.URL{Path: "/DEX/static"}},
			},
			want: true,
		},
		{
			name: "empty path",
			args: args{
				r: &http.Request{URL: &url.URL{Path: ""}},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := omitStatic(tt.args.r); got != tt.want {
				t.Errorf("omitStatic() = %v, want %v", got, tt.want)
			}
		})
	}
}
