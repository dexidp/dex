package main

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

type TestStruct struct {
	Int    int
	String string
	NotMe  string
}

type Test struct {
	Int    int
	String string
	Struct TestStruct
	Hash   string // Crypt hashes are typically of $2a$10$33EMT0cVYVlPy6WAMCLsceLYjWhuHpbz5yuZxu/GAFj03J9Lytjuy, which usually isn't an env...
	Map    map[string]interface{}
}

func TestReplaceEnv(t *testing.T) {
	data := &Test{
		String: "$replace_me",
		// bcrypt hash of the string "password"
		Hash: "$2a$10$33EMT0cVYVlPy6WAMCLsceLYjWhuHpbz5yuZxu/GAFj03J9Lytjuy",
		Struct: TestStruct{
			String: "$me_too",
			NotMe:  "$does_not_exist",
		},
	}

	replacer := func(key string) string {
		switch key {
		case "replace_me":
			return "foo"
		case "me_too":
			return "bar"
		default:
			return ""
		}
	}

	err := replaceEnvKeys(data, replacer)

	if err != nil {
		t.Errorf("Got unexpected error: %s", err)
	}

	expected := &Test{
		String: "foo",
		Struct: TestStruct{String: "bar", NotMe: ""},
		Hash:   "$2a$10$33EMT0cVYVlPy6WAMCLsceLYjWhuHpbz5yuZxu/GAFj03J9Lytjuy",
	}
	if diff := pretty.Compare(data, expected); diff != "" {
		t.Errorf("got!=want: %s", diff)
	}
}
