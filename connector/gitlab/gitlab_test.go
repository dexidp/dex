package gitlab

import "testing"

var nextURLTests = []struct {
	link     string
	expected string
}{
	{"<https://gitlab.com/api/v4/groups?page=2&per_page=20>; rel=\"next\", " +
		"<https://gitlab.com/api/v4/groups?page=1&per_page=20>; rel=\"prev\"; pet=\"cat\", " +
		"<https://gitlab.com/api/v4/groups?page=3&per_page=20>; rel=\"last\"",
		"https://gitlab.com/api/v4/groups?page=2&per_page=20"},
	{"<https://gitlab.com/api/v4/groups?page=3&per_page=20>; rel=\"next\", " +
		"<https://gitlab.com/api/v4/groups?page=2&per_page=20>; rel=\"prev\"; pet=\"dog\", " +
		"<https://gitlab.com/api/v4/groups?page=3&per_page=20>; rel=\"last\"",
		"https://gitlab.com/api/v4/groups?page=3&per_page=20"},
	{"<https://gitlab.com/api/v4/groups?page=3&per_page=20>; rel=\"prev\"; pet=\"bunny\", " +
		"<https://gitlab.com/api/v4/groups?page=3&per_page=20>; rel=\"last\"",
		""},
}

func TestNextURL(t *testing.T) {
	apiURL := "https://gitlab.com/api/v4/groups"
	for _, tt := range nextURLTests {
		apiURL = nextURL(apiURL, tt.link)
		if apiURL != tt.expected {
			t.Errorf("Should have returned %s, got %s", tt.expected, apiURL)
		}
	}
}
