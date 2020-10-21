package groups

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

	// Input identity
	input connector.Identity

	// Output
	wantErr bool
	want    connector.Identity
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
			name: "nogroups",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
		},
		{
			name: "nodiscard",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test"},
			},
		},
		{
			name: "discard1",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"should discard this"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
		},
		{
			name: "discard1mid",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test", "should discard this", "test2"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test", "test2"},
			},
		},
		{
			name: "discard1start",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"should discard this", "test", "test2"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test", "test2"},
			},
		},
		{
			name: "discard1end",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test", "test2", "should discard this"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test", "test2"},
			},
		},
	}

	runTests(t, c, tests)
}

func TestReplace(t *testing.T) {
	c := &Config{
		Actions: []Action{
			{
				Replace: &ReplaceAction{
					Pattern: "(dog|hound)",
					With:    "cat",
				},
			},
		},
	}

	tests := []subtest{
		{
			name: "nogroups",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
		},
		{
			name: "noreplace",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test"},
			},
		},
		{
			name: "replace1",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"dog lovers"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"cat lovers"},
			},
		},
		{
			name: "replace2",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"hound enthusiasts", "dog lovers"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"cat enthusiasts", "cat lovers"},
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
			name: "nogroups",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
		},
		{
			name: "one",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"fun/test"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test"},
			},
		},
		{
			name: "two",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"fun/dogs", "fun/cats"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"dogs", "cats"},
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
			name: "nogroups",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
		},
		{
			name: "one",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"fun/test"},
			},
		},
		{
			name: "two",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"dogs", "cats"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"fun/dogs", "fun/cats"},
			},
		},
	}

	runTests(t, c, tests)
}

func TestSorting(t *testing.T) {
	c := &Config{
		Sorted: true,
	}

	tests := []subtest{
		{
			name: "nogroups",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
		},
		{
			name: "one",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test"},
			},
		},
		{
			name: "two",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"dogs", "cats"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"cats", "dogs"},
			},
		},
		{
			name: "three",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"dogs", "rabbits", "cats"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"cats", "dogs", "rabbits"},
			},
		},
	}

	runTests(t, c, tests)
}

func TestUniquing(t *testing.T) {
	c := &Config{
		Unique: true,
	}

	tests := []subtest{
		{
			name: "nogroups",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
		},
		{
			name: "one",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test"},
			},
		},
		{
			name: "two",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"dogs", "dogs", "cats"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"dogs", "cats"},
			},
		},
		{
			name: "three",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"cats", "dogs", "rabbits", "rabbits", "cats"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"cats", "dogs", "rabbits"},
			},
		},
	}

	runTests(t, c, tests)
}

func TestInject(t *testing.T) {
	c := &Config{
		Inject: []string{
			"birds", "bees", "butterflies",
		},
	}

	tests := []subtest{
		{
			name: "nogroups",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"birds", "bees", "butterflies"},
			},
		},
		{
			name: "one",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"test", "birds", "bees", "butterflies"},
			},
		},
		{
			name: "two",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"dogs", "cats"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups: []string{
					"dogs", "cats", "birds", "bees", "butterflies",
				},
			},
		},
	}

	runTests(t, c, tests)
}

func TestInjectUnique(t *testing.T) {
	c := &Config{
		Inject: []string{
			"birds", "bees", "butterflies",
		},
		Unique: true,
	}

	tests := []subtest{
		{
			name: "nogroups",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"birds", "bees", "butterflies"},
			},
		},
		{
			name: "one",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"bees"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"birds", "bees", "butterflies"},
			},
		},
		{
			name: "two",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"birds", "bats", "butterflies"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups: []string{
					"bats", "birds", "bees", "butterflies",
				},
			},
		},
	}

	runTests(t, c, tests)
}

func TestActionUnique(t *testing.T) {
	c := &Config{
		Actions: []Action{
			{
				Replace: &ReplaceAction{
					Pattern: `\b(cats|dogs|rabbits)\b`,
					With:    "birds",
				},
			},
		},
		Unique: true,
	}

	tests := []subtest{
		{
			name: "nogroups",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
		},
		{
			name: "one",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"cats", "birds"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"birds"},
			},
		},
		{
			name: "two",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"dogs", "rabbits", "birds"},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{"birds"},
			},
		},
	}

	runTests(t, c, tests)
}

func TestMulti(t *testing.T) {
	c := &Config{
		Actions: []Action{
			{
				Discard: "^admin$",
			},
			{
				StripPrefix: "foobar/",
			},
			{
				Replace: &ReplaceAction{
					Pattern: `\b(cats|dogs|rabbits)\b`,
					With:    "birds",
				},
			},
			{
				AddPrefix: "foo/",
			},
		},
		Sorted: true,
		Unique: true,
	}

	tests := []subtest{
		{
			name: "nogroups",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups:        []string{},
			},
		},
		{
			name: "multi",
			input: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups: []string{
					"alternative",
					"admin",
					"verbose",
					"foobar/frobble",
					"cats",
					"rabbits",
					"foobar/dogs",
					"angry cats",
				},
			},
			want: connector.Identity{
				UserID:        "test",
				Username:      "test",
				Email:         "test@example.com",
				EmailVerified: true,
				Groups: []string{
					"foo/alternative",
					"foo/angry birds",
					"foo/birds",
					"foo/frobble",
					"foo/verbose",
				},
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
			got, err := mware.Process(ctx, test.input)
			if err != nil {
				if !test.wantErr {
					t.Fatalf("middleware failed: %v", err)
				}
				return
			}
			if test.wantErr {
				t.Fatal("middleware should have failed")
			}

			if diff := pretty.Compare(test.want, got); diff != "" {
				t.Error(diff)
				return
			}
		})
	}
}
