package claims

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/sirupsen/logrus"

	"github.com/dexidp/dex/connector"
)

type subtest struct {
	// Name of the sub-test
	name string

	// Input claims
	input map[string]interface{}

	// Output
	want map[string]interface{}
}

func TestDiscard(t *testing.T) {
	c := &Config{
		Actions: []Action{
			{
				Discard: `\bd[a-z]+d\b`,
			},
		},
	}

	tests := []subtest{
		{
			name:  "noclaims",
			input: map[string]interface{}{},
			want:  map[string]interface{}{},
		},
		{
			name: "nodiscard",
			input: map[string]interface{}{
				"frobble": 1234,
			},
			want: map[string]interface{}{
				"frobble": 1234,
			},
		},
		{
			name: "discard1",
			input: map[string]interface{}{
				"to discard": 1234,
			},
			want: map[string]interface{}{},
		},
		{
			name: "discard1mid",
			input: map[string]interface{}{
				"test":           true,
				"should discard": 1234,
				"test2":          "foobar",
			},
			want: map[string]interface{}{
				"test":  true,
				"test2": "foobar",
			},
		},
		{
			name: "discard1start",
			input: map[string]interface{}{
				"should discard": 1234,
				"test":           true,
				"test2":          "foobar",
			},
			want: map[string]interface{}{
				"test":  true,
				"test2": "foobar",
			},
		},
		{
			name: "discard1end",
			input: map[string]interface{}{
				"test":           true,
				"test2":          "foobar",
				"should discard": 1234,
			},
			want: map[string]interface{}{
				"test":  true,
				"test2": "foobar",
			},
		},
	}

	runTests(t, c, tests)
}

func TestRename(t *testing.T) {
	c := &Config{
		Actions: []Action{
			{
				Rename: &RenameAction{
					Pattern: "(Dog|Hound)",
					As:      "Cat",
				},
			},
		},
	}

	tests := []subtest{
		{
			name:  "noclaims",
			input: map[string]interface{}{},
			want:  map[string]interface{}{},
		},
		{
			name: "norename",
			input: map[string]interface{}{
				"frobble": 1234,
			},
			want: map[string]interface{}{
				"frobble": 1234,
			},
		},
		{
			name: "rename1",
			input: map[string]interface{}{
				"lovesDogs": true,
			},
			want: map[string]interface{}{
				"lovesCats": true,
			},
		},
		{
			name: "rename2",
			input: map[string]interface{}{
				"test":         true,
				"numberOfDogs": 5,
				"test2":        "foobar",
			},
			want: map[string]interface{}{
				"test":         true,
				"numberOfCats": 5,
				"test2":        "foobar",
			},
		},
		{
			name: "rename3",
			input: map[string]interface{}{
				"likesDogs":      true,
				"numberOfHounds": 7,
			},
			want: map[string]interface{}{
				"likesCats":    true,
				"numberOfCats": 7,
			},
		},
	}

	runTests(t, c, tests)
}

func TestStripPrefix(t *testing.T) {
	c := &Config{
		Actions: []Action{
			{
				StripPrefix: "fun/",
			},
		},
	}

	tests := []subtest{
		{
			name:  "noclaims",
			input: map[string]interface{}{},
			want:  map[string]interface{}{},
		},
		{
			name: "one",
			input: map[string]interface{}{
				"fun/test": "fun/test",
			},
			want: map[string]interface{}{
				"test": "fun/test",
			},
		},
		{
			name: "two",
			input: map[string]interface{}{
				"fun/cats": 5,
				"fun/dogs": 7,
			},
			want: map[string]interface{}{
				"cats": 5,
				"dogs": 7,
			},
		},
	}

	runTests(t, c, tests)
}

func TestAddPrefix(t *testing.T) {
	c := &Config{
		Actions: []Action{
			{
				AddPrefix: "fun/",
			},
		},
	}

	tests := []subtest{
		{
			name:  "noclaims",
			input: map[string]interface{}{},
			want:  map[string]interface{}{},
		},
		{
			name: "one",
			input: map[string]interface{}{
				"test": "fun/test",
			},
			want: map[string]interface{}{
				"fun/test": "fun/test",
			},
		},
		{
			name: "two",
			input: map[string]interface{}{
				"cats": 5,
				"dogs": 7,
			},
			want: map[string]interface{}{
				"fun/cats": 5,
				"fun/dogs": 7,
			},
		},
	}

	runTests(t, c, tests)
}

func TestInject(t *testing.T) {
	c := &Config{
		Inject: map[string]interface{}{
			"example.com/foo":    "bar",
			"example.com/foobar": true,
		},
	}

	tests := []subtest{
		{
			name:  "noclaims",
			input: map[string]interface{}{},
			want: map[string]interface{}{
				"example.com/foo":    "bar",
				"example.com/foobar": true,
			},
		},
		{
			name: "add",
			input: map[string]interface{}{
				"test": "fun/test",
			},
			want: map[string]interface{}{
				"example.com/foo":    "bar",
				"example.com/foobar": true,
				"test":               "fun/test",
			},
		},
		{
			name: "overwrite",
			input: map[string]interface{}{
				"example.com/foo": "fun/test",
			},
			want: map[string]interface{}{
				"example.com/foo":    "bar",
				"example.com/foobar": true,
			},
		},
	}

	runTests(t, c, tests)
}

func runTests(t *testing.T, config *Config, tests []subtest) {
	l := &logrus.Logger{Out: ioutil.Discard, Formatter: &logrus.TextFormatter{}}
	ctx := context.Background()

	mware, err := config.Open(l)
	if err != nil {
		t.Errorf("open middleware: %v", err)
	}

	for _, test := range tests {
		if test.name == "" {
			t.Fatal("subtest has no name")
		}

		t.Run(test.name, func(t *testing.T) {
			identInput := connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
				CustomClaims:  test.input,
			}
			got, err := mware.Process(ctx, identInput)
			if err != nil {
				t.Fatalf("middleware failed: %v", err)
				return
			}

			if diff := pretty.Compare(test.want, got.CustomClaims); diff != "" {
				t.Error(diff)
				return
			}
		})
	}
}
