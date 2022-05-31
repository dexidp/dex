package server

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/Masterminds/sprig/v3"
)

const (
	tmplApproval      = "approval.html"
	tmplLogin         = "login.html"
	tmplPassword      = "password.html"
	tmplOOB           = "oob.html"
	tmplError         = "error.html"
	tmplDevice        = "device.html"
	tmplDeviceSuccess = "device_success.html"
)

var requiredTmpls = []string{
	tmplApproval,
	tmplLogin,
	tmplPassword,
	tmplOOB,
	tmplError,
	tmplDevice,
	tmplDeviceSuccess,
}

type templates struct {
	loginTmpl         *template.Template
	approvalTmpl      *template.Template
	passwordTmpl      *template.Template
	oobTmpl           *template.Template
	errorTmpl         *template.Template
	deviceTmpl        *template.Template
	deviceSuccessTmpl *template.Template
}

type webConfig struct {
	webFS     fs.FS
	logoURL   string
	issuer    string
	theme     string
	issuerURL string
	extra     map[string]string
}

func getFuncMap(c webConfig) (template.FuncMap, error) {
	funcs := sprig.FuncMap()

	issuerURL, err := url.Parse(c.issuerURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing issuerURL: %v", err)
	}

	additionalFuncs := map[string]interface{}{
		"extra":  func(k string) string { return c.extra[k] },
		"issuer": func() string { return c.issuer },
		"logo":   func() string { return c.logoURL },
		"url": func(reqPath, assetPath string) string {
			return relativeURL(issuerURL.Path, reqPath, assetPath)
		},
	}

	for k, v := range additionalFuncs {
		funcs[k] = v
	}

	return funcs, nil
}

// loadWebConfig returns static assets, theme assets, and templates used by the frontend by
// reading the dir specified in the webConfig. If directory is not specified it will
// use the file system specified by webFS.
//
// The directory layout is expected to be:
//
//    ( web directory )
//    |- static
//    |- themes
//    |  |- (theme name)
//    |- templates
//
func loadWebConfig(c webConfig) (http.Handler, http.Handler, *templates, error) {
	// fallback to the default theme if the legacy theme name is provided
	if c.theme == "coreos" || c.theme == "tectonic" {
		c.theme = ""
	}
	if c.theme == "" {
		c.theme = "light"
	}
	if c.issuer == "" {
		c.issuer = "dex"
	}
	if c.logoURL == "" {
		c.logoURL = "theme/logo.png"
	}

	staticFiles, err := fs.Sub(c.webFS, "static")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read static dir: %v", err)
	}
	themeFiles, err := fs.Sub(c.webFS, path.Join("themes", c.theme))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read themes dir: %v", err)
	}

	static := http.FileServer(http.FS(staticFiles))
	theme := http.FileServer(http.FS(themeFiles))

	templates, err := loadTemplates(c, "templates")
	return static, theme, templates, err
}

// loadTemplates parses the expected templates from the provided directory.
func loadTemplates(c webConfig, templatesDir string) (*templates, error) {
	files, err := fs.ReadDir(c.webFS, templatesDir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %v", err)
	}

	filenames := []string{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filenames = append(filenames, path.Join(templatesDir, file.Name()))
	}
	if len(filenames) == 0 {
		return nil, fmt.Errorf("no files in template dir %q", templatesDir)
	}

	funcs, err := getFuncMap(c)
	if err != nil {
		return nil, err
	}

	tmpls, err := template.New("").Funcs(funcs).ParseFS(c.webFS, filenames...)
	if err != nil {
		return nil, fmt.Errorf("parse files: %v", err)
	}
	missingTmpls := []string{}
	for _, tmplName := range requiredTmpls {
		if tmpls.Lookup(tmplName) == nil {
			missingTmpls = append(missingTmpls, tmplName)
		}
	}
	if len(missingTmpls) > 0 {
		return nil, fmt.Errorf("missing template(s): %s", missingTmpls)
	}
	return &templates{
		loginTmpl:         tmpls.Lookup(tmplLogin),
		approvalTmpl:      tmpls.Lookup(tmplApproval),
		passwordTmpl:      tmpls.Lookup(tmplPassword),
		oobTmpl:           tmpls.Lookup(tmplOOB),
		errorTmpl:         tmpls.Lookup(tmplError),
		deviceTmpl:        tmpls.Lookup(tmplDevice),
		deviceSuccessTmpl: tmpls.Lookup(tmplDeviceSuccess),
	}, nil
}

// relativeURL returns the URL of the asset relative to the URL of the request path.
// The serverPath is consulted to trim any prefix due in case it is not listening
// to the root path.
//
// Algorithm:
// 1. Remove common prefix of serverPath and reqPath
// 2. Remove common prefix of assetPath and reqPath
// 3. For each part of reqPath remaining(minus one), go up one level (..)
// 4. For each part of assetPath remaining, append it to result
//
// eg
// server listens at localhost/dex so serverPath is dex
// reqPath is /dex/auth
// assetPath is static/main.css
// relativeURL("/dex", "/dex/auth", "static/main.css") = "../static/main.css"
func relativeURL(serverPath, reqPath, assetPath string) string {
	if u, err := url.ParseRequestURI(assetPath); err == nil && u.Scheme != "" {
		// assetPath points to the external URL, no changes needed
		return assetPath
	}

	splitPath := func(p string) []string {
		res := []string{}
		parts := strings.Split(path.Clean(p), "/")
		for _, part := range parts {
			if part != "" {
				res = append(res, part)
			}
		}
		return res
	}

	stripCommonParts := func(s1, s2 []string) ([]string, []string) {
		min := len(s1)
		if len(s2) < min {
			min = len(s2)
		}

		splitIndex := min
		for i := 0; i < min; i++ {
			if s1[i] != s2[i] {
				splitIndex = i
				break
			}
		}
		return s1[splitIndex:], s2[splitIndex:]
	}

	server, req, asset := splitPath(serverPath), splitPath(reqPath), splitPath(assetPath)

	// Remove common prefix of request path with server path
	_, req = stripCommonParts(server, req)

	// Remove common prefix of request path with asset path
	asset, req = stripCommonParts(asset, req)

	// For each part of the request remaining (minus one) -> go up one level (..)
	// For each part of the asset remaining               -> append it
	var relativeURL string
	for i := 0; i < len(req)-1; i++ {
		relativeURL = path.Join("..", relativeURL)
	}
	relativeURL = path.Join(relativeURL, path.Join(asset...))

	return relativeURL
}

var scopeDescriptions = map[string]string{
	"offline_access": "Have offline access",
	"profile":        "View basic profile information",
	"email":          "View your email address",
	// 'groups' is not a standard OIDC scope, and Dex only returns groups only if the upstream provider does too.
	// This warning is added for convenience to show that the user may expose some sensitive data to the application.
	"groups": "View your groups",
}

type connectorInfo struct {
	ID   string
	Name string
	URL  template.URL
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

func (t *templates) password(r *http.Request, w http.ResponseWriter, postURL, lastUsername, usernamePrompt string, lastWasInvalid bool, backLink string) error {
	data := struct {
		PostURL        string
		BackLink       string
		Username       string
		UsernamePrompt string
		Invalid        bool
		ReqPath        string
	}{postURL, backLink, lastUsername, usernamePrompt, lastWasInvalid, r.URL.Path}
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
