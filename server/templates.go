package server

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	tmplApproval = "approval.html"
	tmplLogin    = "login.html"
	tmplPassword = "password.html"
	tmplOOB      = "oob.html"
	tmplError    = "error.html"
)

type templates struct {
	loginTmpl    *template.Template
	approvalTmpl *template.Template
	passwordTmpl *template.Template
	oobTmpl      *template.Template
	errorTmpl    *template.Template
}

type webConfig struct {
	themeDir     http.FileSystem
	staticDir    http.FileSystem
	templatesDir http.FileSystem
	logoURL      string
	issuer       string
	theme        string
	issuerURL    string
}

func join(basepath string, paths ...string) string {
	u, _ := url.Parse(basepath)
	u.Path = path.Join(append([]string{u.Path}, paths...)...)
	return u.String()
}

// loadTemplates parses the expected templates from the provided directory.
func loadTemplates(c WebConfig, issuerURL string) (*templates, error) {

	if c.Theme == "" {
		c.Theme = "coreos"
	}

	if c.Issuer == "" {
		c.Issuer = "dex"
	}

	if c.LogoURL == "" {
		c.LogoURL = join(issuerURL, "themes", c.Theme, "logo.png")
	}

	hostURL := issuerURL
	if c.HostURL != "" {
		hostURL = c.HostURL
	}

	funcs := template.FuncMap{
		"issuer": func() string { return c.Issuer },
		"logo":   func() string { return c.LogoURL },
		"static": func(s string) string { return join(hostURL, "static", s) },
		"theme":  func(s string) string { return join(hostURL, "themes", c.Theme, s) },
		"lower":  strings.ToLower,
	}

	group := template.New("")

	loginTemplate, err := loadTemplate(c.Dir, tmplLogin, funcs, group)
	if err != nil {
		return nil, err
	}

	approvalTemplate, err := loadTemplate(c.Dir, tmplApproval, funcs, group)
	if err != nil {
		return nil, err
	}

	passwordTemplate, err := loadTemplate(c.Dir, tmplPassword, funcs, group)
	if err != nil {
		return nil, err
	}

	oobTemplate, err := loadTemplate(c.Dir, tmplOOB, funcs, group)
	if err != nil {
		return nil, err
	}

	errorTemplate, err := loadTemplate(c.Dir, tmplError, funcs, group)
	if err != nil {
		return nil, err
	}

	loadTemplate(c.Dir, "header.html", funcs, group)
	loadTemplate(c.Dir, "footer.html", funcs, group)

	return &templates{
		loginTmpl:    loginTemplate,
		approvalTmpl: approvalTemplate,
		passwordTmpl: passwordTemplate,
		oobTmpl:      oobTemplate,
		errorTmpl:    errorTemplate,
	}, nil
}

func loadTemplate(dir http.FileSystem, name string, funcs template.FuncMap, group *template.Template) (*template.Template, error) {
	file, err := dir.Open(filepath.Join("templates", name))
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	buffer.ReadFrom(file)
	contents := buffer.String()

	return group.New(name).Funcs(funcs).Parse(contents)
}

var scopeDescriptions = map[string]string{
	"offline_access": "Have offline access",
	"profile":        "View basic profile information",
	"email":          "View your email address",
}

type connectorInfo struct {
	ID   string
	Name string
	URL  string
}

type byName []connectorInfo

func (n byName) Len() int           { return len(n) }
func (n byName) Less(i, j int) bool { return n[i].Name < n[j].Name }
func (n byName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

func (t *templates) login(w http.ResponseWriter, connectors []connectorInfo) error {
	sort.Sort(byName(connectors))
	data := struct {
		Connectors []connectorInfo
	}{connectors}
	return renderTemplate(w, t.loginTmpl, data)
}

func (t *templates) password(w http.ResponseWriter, postURL, lastUsername, usernamePrompt string, lastWasInvalid, showBacklink bool) error {
	data := struct {
		PostURL        string
		BackLink       bool
		Username       string
		UsernamePrompt string
		Invalid        bool
	}{postURL, showBacklink, lastUsername, usernamePrompt, lastWasInvalid}
	return renderTemplate(w, t.passwordTmpl, data)
}

func (t *templates) approval(w http.ResponseWriter, authReqID, username, clientName string, scopes []string) error {
	accesses := []string{}
	for _, scope := range scopes {
		access, ok := scopeDescriptions[scope]
		if ok {
			accesses = append(accesses, access)
		}
	}
	sort.Strings(accesses)
	data := struct {
		User      string
		Client    string
		AuthReqID string
		Scopes    []string
	}{username, clientName, authReqID, accesses}
	return renderTemplate(w, t.approvalTmpl, data)
}

func (t *templates) oob(w http.ResponseWriter, code string) error {
	data := struct {
		Code string
	}{code}
	return renderTemplate(w, t.oobTmpl, data)
}

func (t *templates) err(w http.ResponseWriter, errCode int, errMsg string) error {
	w.WriteHeader(errCode)
	data := struct {
		ErrType string
		ErrMsg  string
	}{http.StatusText(errCode), errMsg}
	if err := t.errorTmpl.Execute(w, data); err != nil {
		return fmt.Errorf("Error rendering template %s: %s", t.errorTmpl.Name(), err)
	}
	return nil
}

// small io.Writer utility to determine if executing the template wrote to the underlying response writer.
type writeRecorder struct {
	wrote bool
	w     io.Writer
}

func (w *writeRecorder) Write(p []byte) (n int, err error) {
	w.wrote = true
	return w.w.Write(p)
}

func renderTemplate(w http.ResponseWriter, tmpl *template.Template, data interface{}) error {
	wr := &writeRecorder{w: w}
	if err := tmpl.Execute(wr, data); err != nil {
		if !wr.wrote {
			// TODO(ericchiang): replace with better internal server error.
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return fmt.Errorf("Error rendering template %s: %s", tmpl.Name(), err)
	}
	return nil
}
