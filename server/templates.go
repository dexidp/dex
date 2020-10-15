package server

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/markbates/pkger"
)

const (
	tmplApproval      = "approval.html"
	tmplLogin         = "login.html"
	tmplPassword      = "password.html"
	tmplOOB           = "oob.html"
	tmplError         = "error.html"
	tmplDevice        = "device.html"
	tmplDeviceSuccess = "device_success.html"
	tmplHeader        = "header.html"
	tmplFooter        = "footer.html"
)

type templates struct {
	loginTmpl         *template.Template
	approvalTmpl      *template.Template
	passwordTmpl      *template.Template
	oobTmpl           *template.Template
	errorTmpl         *template.Template
	deviceTmpl        *template.Template
	deviceSuccessTmpl *template.Template
}

// loadTemplates parses the expected templates from the provided directory.
func loadTemplates(c WebConfig, issuerPath string) (*templates, error) {
	// fallback to the default theme if the legacy theme name is provided
	if c.Theme == "coreos" || c.Theme == "tectonic" {
		c.Theme = ""
	}
	if c.Theme == "" {
		c.Theme = "light"
	}

	if c.Issuer == "" {
		c.Issuer = "dex"
	}

	if c.LogoURL == "" {
		c.LogoURL = "theme/logo.png"
	}

	hostURL := issuerPath
	if c.HostURL != "" {
		hostURL = c.HostURL
	}

	funcs := template.FuncMap{
		"issuer": func() string { return c.Issuer },
		"logo":   func() string { return c.LogoURL },
		"static": func(assetPath string) string {
			return path.Join(hostURL, "static", assetPath)
		},
		"theme": func(assetPath string) string {
			return path.Join(hostURL, "themes", c.Theme, assetPath)
		},
		"lower": strings.ToLower,
		"extra": func(k string) string { return c.Extra[k] },
	}

	group := template.New("")

	// load all of our templates individually.
	// some http.FilSystem implementations don't implement Readdir

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

	deviceTemplate, err := loadTemplate(c.Dir, tmplDevice, funcs, group)
	if err != nil {
		return nil, err
	}

	deviceSuccessTemplate, err := loadTemplate(c.Dir, tmplDeviceSuccess, funcs, group)
	if err != nil {
		return nil, err
	}

	_, err = loadTemplate(c.Dir, tmplHeader, funcs, group)
	if err != nil {
		// we don't actually care if this template exists
	}

	_, err = loadTemplate(c.Dir, tmplFooter, funcs, group)
	if err != nil {
		// we don't actually care if this template exists
	}

	return &templates{
		loginTmpl:         loginTemplate,
		approvalTmpl:      approvalTemplate,
		passwordTmpl:      passwordTemplate,
		oobTmpl:           oobTemplate,
		errorTmpl:         errorTemplate,
		deviceTmpl:        deviceTemplate,
		deviceSuccessTmpl: deviceSuccessTemplate,
	}, nil
}

// load a template by name from the templates dir
func loadTemplate(dir string, name string, funcs template.FuncMap, group *template.Template) (*template.Template, error) {
	file, err := pkger.Open(filepath.Join(dir, "templates", name))
	if err != nil {
		return nil, err
	}

	defer file.Close()

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
	Type string
}

type byName []connectorInfo

func (n byName) Len() int           { return len(n) }
func (n byName) Less(i, j int) bool { return n[i].Name < n[j].Name }
func (n byName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

func (t *templates) device(r *http.Request, w http.ResponseWriter, postURL string, userCode string, lastWasInvalid bool) error {
	if lastWasInvalid {
		w.WriteHeader(http.StatusBadRequest)
	}
	data := struct {
		PostURL  string
		UserCode string
		Invalid  bool
		ReqPath  string
	}{postURL, userCode, lastWasInvalid, r.URL.Path}
	return renderTemplate(w, t.deviceTmpl, data)
}

func (t *templates) deviceSuccess(r *http.Request, w http.ResponseWriter, clientName string) error {
	data := struct {
		ClientName string
		ReqPath    string
	}{clientName, r.URL.Path}
	return renderTemplate(w, t.deviceSuccessTmpl, data)
}

func (t *templates) login(r *http.Request, w http.ResponseWriter, connectors []connectorInfo) error {
	sort.Sort(byName(connectors))
	data := struct {
		Connectors []connectorInfo
		ReqPath    string
	}{connectors, r.URL.Path}
	return renderTemplate(w, t.loginTmpl, data)
}

func (t *templates) password(r *http.Request, w http.ResponseWriter, postURL, lastUsername, usernamePrompt string, lastWasInvalid, showBacklink bool) error {
	data := struct {
		PostURL        string
		BackLink       bool
		Username       string
		UsernamePrompt string
		Invalid        bool
		ReqPath        string
	}{postURL, showBacklink, lastUsername, usernamePrompt, lastWasInvalid, r.URL.Path}
	return renderTemplate(w, t.passwordTmpl, data)
}

func (t *templates) approval(r *http.Request, w http.ResponseWriter, authReqID, username, clientName string, scopes []string) error {
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
		ReqPath   string
	}{username, clientName, authReqID, accesses, r.URL.Path}
	return renderTemplate(w, t.approvalTmpl, data)
}

func (t *templates) oob(r *http.Request, w http.ResponseWriter, code string) error {
	data := struct {
		Code    string
		ReqPath string
	}{code, r.URL.Path}
	return renderTemplate(w, t.oobTmpl, data)
}

func (t *templates) err(r *http.Request, w http.ResponseWriter, errCode int, errMsg string) error {
	w.WriteHeader(errCode)
	data := struct {
		ErrType string
		ErrMsg  string
		ReqPath string
	}{http.StatusText(errCode), errMsg, r.URL.Path}
	if err := t.errorTmpl.Execute(w, data); err != nil {
		return fmt.Errorf("rendering template %s failed: %s", t.errorTmpl.Name(), err)
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
		return fmt.Errorf("rendering template %s failed: %s", tmpl.Name(), err)
	}
	return nil
}
