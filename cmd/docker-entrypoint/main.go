// Package main provides a utility program to launch the Dex container process with an optional
// templating step (provided by gomplate).
//
// This was originally written as a shell script, but we rewrote it as a Go program so that it could
// run as a raw binary in a distroless container.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func main() {
	// Note that this docker-entrypoint program is args[0], and it is provided with the true process
	// args.
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("error: no args passed to entrypoint")
		os.Exit(1)
	}

	if err := run(args, realExec, realWhich, realGomplate); err != nil {
		fmt.Println("error:", err.Error())
		os.Exit(1)
	}
}

func realExec(args ...string) error {
	argv0, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("cannot lookup path for command %s: %w", args[0], err)
	}

	if err := syscall.Exec(argv0, args, os.Environ()); err != nil {
		return fmt.Errorf("cannot exec command %s (%q): %w", args, argv0, err)
	}

	return nil
}

func realWhich(path string) string {
	fullPath, err := exec.LookPath(path)
	if err != nil {
		return ""
	}
	return fullPath
}

func realGomplate(path string) (string, error) {
	tmpFile, err := os.CreateTemp("/tmp", "dex.config.yaml-*")
	if err != nil {
		return "", fmt.Errorf("cannot create temp file: %w", err)
	}

	cmd := exec.Command("gomplate", "-f", path, "-o", tmpFile.Name())
	// TODO(nabokihms): Workaround to run gomplate from a non-root directory in distroless images
	//   gomplate tries to access CWD on start, see: https://github.com/hairyhenderson/gomplate/pull/2202
	cmd.Dir = "/etc/dex"

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing gomplate: %w, (output: %q)", err, string(output))
	}

	return tmpFile.Name(), nil
}

func run(args []string, execFunc func(...string) error, whichFunc func(string) string, gomplateFunc func(string) (string, error)) error {
	if args[0] != "dex" && args[0] != whichFunc("dex") {
		return execFunc(args...)
	}

	if args[1] != "serve" {
		return execFunc(args...)
	}

	newArgs := []string{}
	for _, tplCandidate := range args {
		if hasSuffixes(tplCandidate, ".tpl", ".tmpl", ".yaml") {
			fileName, err := gomplateFunc(tplCandidate)
			if err != nil {
				return err
			}

			newArgs = append(newArgs, fileName)
		} else {
			newArgs = append(newArgs, tplCandidate)
		}
	}

	return execFunc(newArgs...)
}

func hasSuffixes(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}
