package main

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

type TestStruct struct {
	Int    int
	String string
}

type Test struct {
	Int    int
	String string
	Struct TestStruct
	Map    map[string]interface{}
}

func TestReplaceEnv(t *testing.T) {
	data := &Test{
		String: "$replace_me",
		Struct: TestStruct{
			String: "$me_too",
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

	expected := &Test{String: "foo", Struct: TestStruct{String: "bar"}}
	if diff := pretty.Compare(data, expected); diff != "" {
		t.Errorf("got!=want: %s", diff)
	}
}
