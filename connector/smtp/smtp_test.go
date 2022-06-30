package smtp

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"testing"
	"time"

	"github.com/dexidp/dex/connector"
	emsmtp "github.com/emersion/go-smtp"
	"github.com/sirupsen/logrus"
)

func TestSMTP(t *testing.T) {
	l := runServer(t)
	c := &Config{
		Host: l.Addr().String(),
	}
	// Closing the listentin port we got from runServer will cause the
	// Serve() to return, and the goroutine to exit.
	defer l.Close()

	logger := logrus.New()

	ctr, err := c.Open("test", logger)
	if err != nil {
		t.Fatal(err)
	}
	sc := ctr.(*smtpConnector)

	if sc.Prompt() != "" {
		t.Error("expected empty prompt")
	}

	_, valid, err := sc.Login(context.Background(), connector.Scopes{}, "user", "badpass")
	if err != nil {
		t.Error("err where there should be none: ", err)
	}
	if valid != false {
		t.Error("got unexpected valid")
	}

	ctr, err = c.Open("test", logger)
	if err != nil {
		t.Fatal(err)
	}
	sc = ctr.(*smtpConnector)
	id, valid, err := sc.Login(context.Background(), connector.Scopes{}, "user", "goodpass")
	if err != nil {
		t.Fatal(err)
	}
	if valid != true {
		t.Error("got unexpected valid false")
	}
	if id.Username != "user" {
		t.Error("wrong username")
	}
}

// Scaffolding for a fake SMTP server starts here

// The Backend implements SMTP server methods.
type Backend struct {
}

// Login handles a login command with username and password.
func (bkd *Backend) Login(state *emsmtp.ConnectionState, username, password string) (emsmtp.Session, error) {
	if username != "user" || password != "goodpass" {
		return nil, errors.New("Invalid username or password")
	}
	return &Session{}, nil
}

// AnonymousLogin requires clients to authenticate using SMTP AUTH before sending emails
func (bkd *Backend) AnonymousLogin(state *emsmtp.ConnectionState) (emsmtp.Session, error) {
	return nil, emsmtp.ErrAuthRequired
}

// A Session is returned after successful login.
type Session struct{}

func (s *Session) Mail(from string, opts emsmtp.MailOptions) error {
	log.Println("Mail from:", from)
	return nil
}

func (s *Session) Rcpt(to string) error {
	log.Println("Rcpt to:", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	if b, err := ioutil.ReadAll(r); err != nil {
		return err
	} else {
		log.Println("Data:", string(b))
	}
	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}

func runServer(t *testing.T) net.Listener {
	be := &Backend{}

	s := emsmtp.NewServer(be)

	s.Domain = "localhost"
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true

	// :0 says, "use any free port for the listening socket"
	// then we need to return the socket to the caller, so they
	// know where to find the fake server.
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	s.Addr = l.Addr().String()

	t.Log("Starting server at", s.Addr)
	go func() {
		s.Serve(l)
		t.Log("Fake SMTP server exited.")
	}()

	return l
}
