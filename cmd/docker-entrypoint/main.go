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

	if err := run(args, realExec, realWhich); err != nil {
		fmt.Println("error:", err.Error())
		os.Exit(1)
	}
}

func realExec(fork bool, args ...string) error {
	if fork {
		if output, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
			return fmt.Errorf("cannot fork/exec command %s: %w (output: %q)", args, err, string(output))
		}
		return nil
	}

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

func run(args []string, execFunc func(bool, ...string) error, whichFunc func(string) string) error {
	if args[0] != "dex" && args[0] != whichFunc("dex") {
		return execFunc(false, args...)
	}

	if args[1] != "serve" {
		return execFunc(false, args...)
	}

	newArgs := []string{}
	for _, tplCandidate := range args {
		if hasSuffixes(tplCandidate, ".tpl", ".tmpl", ".yaml") {
			tmpFile, err := os.CreateTemp("/tmp", "dex.config.yaml-*")
			if err != nil {
				return fmt.Errorf("cannot create temp file: %w", err)
			}

			if err := execFunc(true, "gomplate", "-f", tplCandidate, "-o", tmpFile.Name()); err != nil {
				return err
			}

			newArgs = append(newArgs, tmpFile.Name())
		} else {
			newArgs = append(newArgs, tplCandidate)
		}
	}

	return execFunc(false, newArgs...)
}

func hasSuffixes(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}
