package functional

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

var connConfigExample = []byte(`[
	{
		"type": "local",
		"id": "local",
		"passwordInfos": [
			{
				"userId":"elroy-id",
				"passwordPlaintext": "bones"
			},
			{
				"userId":"penny",
				"passwordPlaintext": "kibble"
			}
		]
	}
]`)

func TestDexctlCommands(t *testing.T) {
	if strings.HasPrefix(dsn, "sqlite3://") {
		t.Skip("only test dexctl conmand with postgres")
	}
	tempFile, err := ioutil.TempFile("", "dexctl_functional_tests_")
	if err != nil {
		t.Fatal(err)
	}
	connConfig := tempFile.Name()
	defer os.Remove(connConfig)
	if _, err := tempFile.Write(connConfigExample); err != nil {
		t.Fatal(err)
	}

	tempFile.Close()

	tests := []struct {
		args   []string
		env    []string
		expErr bool
		stdin  io.Reader
	}{
		{
			args: []string{"new-client", "https://example.com/callback"},
			env:  []string{"DEXCTL_DB_URL=" + dsn},
		},
		{
			args: []string{"new-client", "--db-url", dsn, "https://example.com/callback"},
		},
		{
			args: []string{"set-connector-configs", connConfig},
			env:  []string{"DEXCTL_DB_URL=" + dsn},
		},
		{
			args: []string{"set-connector-configs", "--db-url", dsn, connConfig},
		},
		{
			args:  []string{"set-connector-configs", "--db-url", dsn, "-"},
			stdin: bytes.NewReader(connConfigExample),
		},
		{
			args:  []string{"set-connector-configs", "-"},
			env:   []string{"DEXCTL_DB_URL=" + dsn},
			stdin: bytes.NewReader(connConfigExample),
		},
	}

	for _, tt := range tests {
		cmd := exec.Command("../bin/dexctl", tt.args...)
		cmd.Stdin = tt.stdin
		cmd.Env = tt.env
		out, err := cmd.CombinedOutput()
		if !tt.expErr && err != nil {
			t.Errorf("cmd 'dexctl %s' failed: %v %s", strings.Join(tt.args, " "), err, out)
		} else if tt.expErr && err == nil {
			t.Errorf("expected cmd 'dexctl %s' to fail", strings.Join(tt.args, " "))
		}
	}
}
