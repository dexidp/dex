// Copyright 2016 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// Author: Marc Berhault (marc@cockroachlabs.com)

// Package testserver provides helpers to run a cockroach binary within tests.
// It automatically downloads the latest cockroach binary for your platform
// (Linux-amd64 and Darwin-amd64 only for now), or attempts to run "cockroach"
// from your PATH.
//
// A normal invocation is (check err every time):
// ts, err := testserver.NewTestServer()
// err = ts.Start()
// defer ts.Stop()
// url := ts.PGURL()
//
// To use, run as follows:
//   import "github.com/cockroachdb/cockroach-go/testserver"
//   import "testing"
//   import "time"
//
//   func TestRunServer(t *testing.T) {
//      ts, err := testserver.NewTestServer()
//      if err != nil {
//        t.Fatal(err)
//      }
//      err := ts.Start()
//      if err != nil {
//        t.Fatal(err)
//      }
//      defer ts.Stop()
//
//      url := ts.PGURL()
//      if url != nil {
//        t.FatalF("url not found")
//      }
//      t.Logf("URL: %s", url.String())
//
//      db, err := sql.Open("postgres", url.String())
//      if err != nil {
//        t.Fatal(err)
//      }
//    }
package testserver

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

var sqlURLRegexp = regexp.MustCompile("sql:\\s+(postgresql:.+)\n")

const (
	stateNew     = iota
	stateRunning = iota
	stateStopped = iota
	stateFailed  = iota

	socketPort     = 26257
	socketFileBase = ".s.PGSQL"
)

// TestServer is a helper to run a real cockroach node.
type TestServer struct {
	mu        sync.RWMutex
	state     int
	baseDir   string
	pgURL     *url.URL
	cmd       *exec.Cmd
	args      []string
	stdout    string
	stderr    string
	stdoutBuf logWriter
	stderrBuf logWriter
}

// NewDBForTest creates a new CockroachDB TestServer instance and
// opens a SQL database connection to it. Returns a sql *DB instance a
// shutdown function. The caller is responsible for executing the
// returned shutdown function on exit.
func NewDBForTest(t *testing.T) (*sql.DB, func()) {
	return NewDBForTestWithDatabase(t, "")
}

// NewDBForTestWithDatabase creates a new CockroachDB TestServer
// instance and opens a SQL database connection to it. If database is
// specified, the returned connection will explicitly connect to
// it. Returns a sql *DB instance a shutdown function. The caller is
// responsible for executing the returned shutdown function on exit.
func NewDBForTestWithDatabase(t *testing.T, database string) (*sql.DB, func()) {
	ts, err := NewTestServer()
	if err != nil {
		t.Fatal(err)
	}

	err = ts.Start()
	if err != nil {
		t.Fatal(err)
	}

	url := ts.PGURL()
	if url == nil {
		t.Fatalf("url not found")
	}
	if len(database) > 0 {
		url.Path = database
	}

	db, err := sql.Open("postgres", url.String())
	if err != nil {
		t.Fatal(err)
	}

	ts.WaitForInit(db)

	return db, func() {
		_ = db.Close()
		ts.Stop()
	}
}

// NewTestServer creates a new TestServer, but does not start it.
// The cockroach binary for your OS and ARCH is downloaded automatically.
// If the download fails, we attempt just call "cockroach", hoping it is
// found in your path.
func NewTestServer() (*TestServer, error) {
	cockroachBinary, err := downloadLatestBinary()
	if err == nil {
		log.Printf("Using automatically-downloaded binary: %s", cockroachBinary)
	} else {
		log.Printf("Attempting to use cockroach binary from your PATH")
		cockroachBinary = "cockroach"
	}

	// Force "/tmp/" so avoid OSX's really long temp directory names
	// which get us over the socket filename length limit.
	baseDir, err := ioutil.TempDir("/tmp", "cockroach-testserver")
	if err != nil {
		return nil, fmt.Errorf("could not create temp directory: %s", err)
	}

	logDir := filepath.Join(baseDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create logs directory: %s: %s", logDir, err)
	}

	options := url.Values{
		"host": []string{baseDir},
	}
	pgurl := &url.URL{
		Scheme:   "postgres",
		User:     url.User("root"),
		Host:     fmt.Sprintf(":%d", socketPort),
		RawQuery: options.Encode(),
	}
	socketPath := filepath.Join(baseDir, fmt.Sprintf("%s.%d", socketFileBase, socketPort))

	args := []string{
		cockroachBinary,
		"start",
		"--logtostderr",
		"--insecure",
		"--port=0",
		"--http-port=0",
		"--socket=" + socketPath,
		"--store=" + baseDir,
	}

	ts := &TestServer{
		baseDir: baseDir,
		pgURL:   pgurl,
		args:    args,
		stdout:  filepath.Join(logDir, "cockroach.stdout"),
		stderr:  filepath.Join(logDir, "cockroach.stderr"),
	}
	return ts, nil
}

// Stdout returns the entire contents of the process' stdout.
func (ts *TestServer) Stdout() string {
	return ts.stdoutBuf.String()
}

// Stderr returns the entire contents of the process' stderr.
func (ts *TestServer) Stderr() string {
	return ts.stderrBuf.String()
}

// PGURL returns the postgres connection URL to reach the started
// cockroach node.
// It loops until the expected unix socket file exists.
// This does not timeout, relying instead on test timeouts.
func (ts *TestServer) PGURL() *url.URL {
	socketPath := filepath.Join(ts.baseDir, fmt.Sprintf("%s.%d", socketFileBase, socketPort))
	for {
		if _, err := os.Stat(socketPath); err == nil {
			return ts.pgURL
		}
		time.Sleep(time.Millisecond * 10)
	}
	return nil
}

// WaitForInit repeatedly looks up the list of databases until
// the "system" database exists. It ignores all errors as we are
// waiting for the process to start and complete initialization.
// This does not timeout, relying instead on test timeouts.
func (ts *TestServer) WaitForInit(db *sql.DB) {
	for {
		// We issue a query that fails both on connection errors and on the
		// system database not existing.
		if _, err := db.Query("SHOW DATABASES"); err == nil {
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
}

// Start runs the process, returning an error on any problems,
// including being unable to start, but not unexpected failure.
// It should only be called once in the lifetime of a TestServer object.
func (ts *TestServer) Start() error {
	ts.mu.Lock()
	if ts.state != stateNew {
		ts.mu.Unlock()
		return errors.New("Start() can only be called once")
	}
	ts.state = stateRunning
	ts.mu.Unlock()

	ts.cmd = exec.Command(ts.args[0], ts.args[1:]...)
	ts.cmd.Env = []string{"COCKROACH_MAX_OFFSET=1ns"}

	if len(ts.stdout) > 0 {
		wr, err := newFileLogWriter(ts.stdout)
		if err != nil {
			return fmt.Errorf("unable to open file %s: %s", ts.stdout, err)
		}
		ts.stdoutBuf = wr
	}
	ts.cmd.Stdout = ts.stdoutBuf

	if len(ts.stderr) > 0 {
		wr, err := newFileLogWriter(ts.stderr)
		if err != nil {
			return fmt.Errorf("unable to open file %s: %s", ts.stderr, err)
		}
		ts.stderrBuf = wr
	}
	ts.cmd.Stderr = ts.stderrBuf

	for k, v := range defaultEnv() {
		ts.cmd.Env = append(ts.cmd.Env, k+"="+v)
	}

	err := ts.cmd.Start()
	if ts.cmd.Process != nil {
		log.Printf("process %d started: %s", ts.cmd.Process.Pid, strings.Join(ts.args, " "))
	}
	if err != nil {
		log.Printf(err.Error())
		ts.stdoutBuf.Close()
		ts.stderrBuf.Close()

		ts.mu.Lock()
		ts.state = stateFailed
		ts.mu.Unlock()

		return fmt.Errorf("failure starting process: %s", err)
	}

	go func() {
		ts.cmd.Wait()

		ts.stdoutBuf.Close()
		ts.stderrBuf.Close()

		ps := ts.cmd.ProcessState
		sy := ps.Sys().(syscall.WaitStatus)

		log.Printf("Process %d exited with status %d", ps.Pid(), sy.ExitStatus())
		log.Printf(ps.String())

		ts.mu.Lock()
		if sy.ExitStatus() == 0 {
			ts.state = stateStopped
		} else {
			ts.state = stateFailed
		}
		ts.mu.Unlock()
	}()

	return nil
}

// Stop kills the process if it is still running and cleans its directory.
// It should only be called once in the lifetime of a TestServer object.
// Logs fatal if the process has already failed.
func (ts *TestServer) Stop() {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if ts.state == stateNew {
		log.Fatal("Stop() called, but Start() was never called")
	}
	if ts.state == stateFailed {
		log.Fatalf("Stop() called, but process exited unexpectedly. Stdout:\n%s\nStderr:\n%s\n",
			ts.Stdout(), ts.Stderr())
		return
	}

	if ts.state != stateStopped {
		// Only call kill if not running. It could have exited properly.
		ts.cmd.Process.Kill()
	}

	// Only cleanup on intentional stops.
	_ = os.RemoveAll(ts.baseDir)
}

type logWriter interface {
	Write(p []byte) (n int, err error)
	String() string
	Len() int64
	Close()
}

type fileLogWriter struct {
	filename string
	file     *os.File
}

func newFileLogWriter(file string) (*fileLogWriter, error) {
	f, err := os.Create(file)
	if err != nil {
		return nil, err
	}

	return &fileLogWriter{
		filename: file,
		file:     f,
	}, nil
}

func (w fileLogWriter) Close() {
	w.file.Close()
}

func (w fileLogWriter) Write(p []byte) (n int, err error) {
	return w.file.Write(p)
}

func (w fileLogWriter) String() string {
	b, err := ioutil.ReadFile(w.filename)
	if err == nil {
		return string(b)
	}
	return ""
}

func (w fileLogWriter) Len() int64 {
	s, err := os.Stat(w.filename)
	if err == nil {
		return s.Size()
	}
	return 0
}

func defaultEnv() map[string]string {
	vars := map[string]string{}
	u, err := user.Current()
	if err == nil {
		if _, ok := vars["USER"]; !ok {
			vars["USER"] = u.Username
		}
		if _, ok := vars["UID"]; !ok {
			vars["UID"] = u.Uid
		}
		if _, ok := vars["GID"]; !ok {
			vars["GID"] = u.Gid
		}
		if _, ok := vars["HOME"]; !ok {
			vars["HOME"] = u.HomeDir
		}
	}
	if _, ok := vars["PATH"]; !ok {
		vars["PATH"] = os.Getenv("PATH")
	}
	return vars
}
