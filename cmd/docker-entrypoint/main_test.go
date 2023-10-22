package main

import (
	"strings"
	"testing"
)

type execArgs struct {
	fork        bool
	argPrefixes []string
}

func TestRun(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		execReturns  error
		whichReturns string
		wantExecArgs []execArgs
		wantErr      error
	}{
		{
			name:         "executable not dex",
			args:         []string{"tuna", "fish"},
			wantExecArgs: []execArgs{{fork: false, argPrefixes: []string{"tuna", "fish"}}},
		},
		{
			name:         "executable is full path to dex",
			args:         []string{"/usr/local/bin/dex", "marshmallow", "zelda"},
			whichReturns: "/usr/local/bin/dex",
			wantExecArgs: []execArgs{{fork: false, argPrefixes: []string{"/usr/local/bin/dex", "marshmallow", "zelda"}}},
		},
		{
			name:         "command is not serve",
			args:         []string{"dex", "marshmallow", "zelda"},
			wantExecArgs: []execArgs{{fork: false, argPrefixes: []string{"dex", "marshmallow", "zelda"}}},
		},
		{
			name:         "no templates",
			args:         []string{"dex", "serve", "config.yaml.not-a-template"},
			wantExecArgs: []execArgs{{fork: false, argPrefixes: []string{"dex", "serve", "config.yaml.not-a-template"}}},
		},
		{
			name:         "no templates",
			args:         []string{"dex", "serve", "config.yaml.not-a-template"},
			wantExecArgs: []execArgs{{fork: false, argPrefixes: []string{"dex", "serve", "config.yaml.not-a-template"}}},
		},
		{
			name: ".tpl template",
			args: []string{"dex", "serve", "config.tpl"},
			wantExecArgs: []execArgs{
				{fork: true, argPrefixes: []string{"gomplate", "-f", "config.tpl", "-o", "/tmp/dex.config.yaml-"}},
				{fork: false, argPrefixes: []string{"dex", "serve", "/tmp/dex.config.yaml-"}},
			},
		},
		{
			name: ".tmpl template",
			args: []string{"dex", "serve", "config.tmpl"},
			wantExecArgs: []execArgs{
				{fork: true, argPrefixes: []string{"gomplate", "-f", "config.tmpl", "-o", "/tmp/dex.config.yaml-"}},
				{fork: false, argPrefixes: []string{"dex", "serve", "/tmp/dex.config.yaml-"}},
			},
		},
		{
			name: ".yaml template",
			args: []string{"dex", "serve", "some/path/config.yaml"},
			wantExecArgs: []execArgs{
				{fork: true, argPrefixes: []string{"gomplate", "-f", "some/path/config.yaml", "-o", "/tmp/dex.config.yaml-"}},
				{fork: false, argPrefixes: []string{"dex", "serve", "/tmp/dex.config.yaml-"}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotExecForks []bool
			var gotExecArgs [][]string
			fakeExec := func(fork bool, args ...string) error {
				gotExecForks = append(gotExecForks, fork)
				gotExecArgs = append(gotExecArgs, args)
				return test.execReturns
			}

			fakeWhich := func(_ string) string { return test.whichReturns }

			gotErr := run(test.args, fakeExec, fakeWhich)
			if (test.wantErr == nil) != (gotErr == nil) {
				t.Errorf("wanted error %s, got %s", test.wantErr, gotErr)
			}
			if !execArgsMatch(test.wantExecArgs, gotExecForks, gotExecArgs) {
				t.Errorf("wanted exec args %+v, got %+v %+v", test.wantExecArgs, gotExecForks, gotExecArgs)
			}
		})
	}
}

func execArgsMatch(wantExecArgs []execArgs, gotForks []bool, gotExecArgs [][]string) bool {
	if len(wantExecArgs) != len(gotForks) {
		return false
	}

	for i := range wantExecArgs {
		if wantExecArgs[i].fork != gotForks[i] {
			return false
		}
		for j := range wantExecArgs[i].argPrefixes {
			if !strings.HasPrefix(gotExecArgs[i][j], wantExecArgs[i].argPrefixes[j]) {
				return false
			}
		}
	}

	return true
}
