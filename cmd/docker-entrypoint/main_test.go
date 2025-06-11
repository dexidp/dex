package main

import (
	"strings"
	"testing"
)

type execArgs struct {
	gomplate    bool
	argPrefixes []string
}

func TestRun(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		execReturns  error
		whichReturns string
		wantExecArgs execArgs
		wantErr      error
	}{
		{
			name:         "executable not dex",
			args:         []string{"tuna", "fish"},
			wantExecArgs: execArgs{gomplate: false, argPrefixes: []string{"tuna", "fish"}},
		},
		{
			name:         "executable is full path to dex",
			args:         []string{"/usr/local/bin/dex", "marshmallow", "zelda"},
			whichReturns: "/usr/local/bin/dex",
			wantExecArgs: execArgs{gomplate: false, argPrefixes: []string{"/usr/local/bin/dex", "marshmallow", "zelda"}},
		},
		{
			name:         "command is not serve",
			args:         []string{"dex", "marshmallow", "zelda"},
			wantExecArgs: execArgs{gomplate: false, argPrefixes: []string{"dex", "marshmallow", "zelda"}},
		},
		{
			name:         "no templates",
			args:         []string{"dex", "serve", "config.yaml.not-a-template"},
			wantExecArgs: execArgs{gomplate: false, argPrefixes: []string{"dex", "serve", "config.yaml.not-a-template"}},
		},
		{
			name:         "no templates",
			args:         []string{"dex", "serve", "config.yaml.not-a-template"},
			wantExecArgs: execArgs{gomplate: false, argPrefixes: []string{"dex", "serve", "config.yaml.not-a-template"}},
		},
		{
			name:         ".tpl template",
			args:         []string{"dex", "serve", "config.tpl"},
			wantExecArgs: execArgs{gomplate: true, argPrefixes: []string{"dex", "serve", "/tmp/dex.config.yaml-"}},
		},
		{
			name:         ".tmpl template",
			args:         []string{"dex", "serve", "config.tmpl"},
			wantExecArgs: execArgs{gomplate: true, argPrefixes: []string{"dex", "serve", "/tmp/dex.config.yaml-"}},
		},
		{
			name:         ".yaml template",
			args:         []string{"dex", "serve", "some/path/config.yaml"},
			wantExecArgs: execArgs{gomplate: true, argPrefixes: []string{"dex", "serve", "/tmp/dex.config.yaml-"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotExecArgs []string
			var runsGomplate bool

			fakeExec := func(args ...string) error {
				gotExecArgs = append(args, gotExecArgs...)
				return test.execReturns
			}

			fakeWhich := func(_ string) string { return test.whichReturns }

			fakeGomplate := func(file string) (string, error) {
				runsGomplate = true
				return "/tmp/dex.config.yaml-", nil
			}

			gotErr := run(test.args, fakeExec, fakeWhich, fakeGomplate)
			if (test.wantErr == nil) != (gotErr == nil) {
				t.Errorf("wanted error %s, got %s", test.wantErr, gotErr)
			}

			if !execArgsMatch(test.wantExecArgs, runsGomplate, gotExecArgs) {
				t.Errorf("wanted exec args %+v (running gomplate: %+v), got %+v (running gomplate: %+v)",
					test.wantExecArgs.argPrefixes, test.wantExecArgs.gomplate, gotExecArgs, runsGomplate)
			}
		})
	}
}

func execArgsMatch(wantExecArgs execArgs, gomplate bool, gotExecArgs []string) bool {
	if wantExecArgs.gomplate != gomplate {
		return false
	}
	for i := range wantExecArgs.argPrefixes {
		if !strings.HasPrefix(gotExecArgs[i], wantExecArgs.argPrefixes[i]) {
			return false
		}
	}
	return true
}
