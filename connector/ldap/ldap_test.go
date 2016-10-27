package ldap

import "testing"

func TestEscapeFilter(t *testing.T) {
	tests := []struct {
		val  string
		want string
	}{
		{"Five*Star", "Five\\2aStar"},
		{"c:\\File", "c:\\5cFile"},
		{"John (2nd)", "John \\282nd\\29"},
		{string([]byte{0, 0, 0, 4}), "\\00\\00\\00\\04"},
		{"Chloé", "Chlo\\c3\\a9"},
	}

	for _, tc := range tests {
		got := escapeFilter(tc.val)
		if tc.want != got {
			t.Errorf("value %q want=%q, got=%q", tc.val, tc.want, got)
		}
	}
}
