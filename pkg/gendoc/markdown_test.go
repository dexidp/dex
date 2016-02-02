package gendoc

import (
	"testing"

	"github.com/kylelemons/godebug/diff"
)

func TestToAnchor(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{"foo", "foo"},
		{"foo bar", "foo-bar"},
		{"POST /foo/{id}", "post-fooid"},
	}

	for _, tt := range tests {
		if got := toAnchor(tt.s); got != tt.want {
			t.Errorf("toAnchor(%q): want=%q, got=%q", tt.s, tt.want, got)
		}
	}
}

func TestToJSON(t *testing.T) {
	tests := []struct {
		s    Schema
		want string
	}{
		{
			s: Schema{
				Name: "UsersResponse",
				Type: "object",
				Children: []Schema{
					{
						Name: "nextPageToken",
						Type: "string",
					},
					{
						Name: "users",
						Type: "array",
						Children: []Schema{
							{
								Ref: "User",
							},
						},
					},
				},
			},
			want: "```" + `
{
    nextPageToken: string,
    users: [
        User
    ]
}
` + "```",
		},
	}

	for i, tt := range tests {
		got := tt.s.toJSON()
		if d := diff.Diff(got, tt.want); d != "" {
			t.Errorf("case %d: want != got: %s", i, d)
		}
	}
}
