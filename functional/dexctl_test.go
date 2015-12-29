package functional

import (
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
	}{
		{
			args: []string{"new-client", "https://example.com/callback"},
			env:  []string{"DEXCTL_DB_URL=" + dsn, "DEXCTL_DB=" + storage},
		},
		{
			args: []string{"new-client", "--db", storage, "--db-url", dsn, "https://example.com/callback"},
		},
		{
			args: []string{"set-connector-configs", connConfig},
			env:  []string{"DEXCTL_DB_URL=" + dsn, "DEXCTL_DB=" + storage},
		},
		{
			args: []string{"set-connector-configs", "--db", storage, "--db-url", dsn, connConfig},
		},
	}

	for _, tt := range tests {
		cmd := exec.Command("../bin/dexctl", tt.args...)
		cmd.Env = tt.env
		out, err := cmd.CombinedOutput()
		if !tt.expErr && err != nil {
			t.Errorf("cmd 'dexctl %s' failed: %v %s", strings.Join(tt.args, " "), err, out)
		} else if tt.expErr && err == nil {
			t.Errorf("expected cmd 'dexctl %s' to fail", strings.Join(tt.args, " "))
		}
	}
}
