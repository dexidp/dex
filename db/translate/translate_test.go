package translate

import "testing"

func TestPostgresToSQLite(t *testing.T) {

	tests := []struct {
		query string
		want  string
	}{
		{"SELECT * FROM foo", "SELECT * FROM foo"},
		{"SELECT * FROM %s", "SELECT * FROM %s"},
		{"SELECT * FROM foo WHERE is_admin=true", "SELECT * FROM foo WHERE is_admin=1"},
		{"SELECT * FROM foo WHERE is_admin=true;", "SELECT * FROM foo WHERE is_admin=1;"},
		{"SELECT * FROM foo WHERE is_admin=$10", "SELECT * FROM foo WHERE is_admin=?"},
		{"SELECT * FROM foo WHERE is_admin=$10;", "SELECT * FROM foo WHERE is_admin=?;"},
		{"SELECT * FROM foo WHERE name=$1 AND is_admin=$2;", "SELECT * FROM foo WHERE name=? AND is_admin=?;"},
		{"$1", "?"},
		{"$", "$"},
	}

	for _, tt := range tests {
		got := PostgresToSQLite(tt.query)
		if got != tt.want {
			t.Errorf("PostgresToSQLite(%q): want=%q, got=%q", tt.query, tt.want, got)
		}
	}
}
