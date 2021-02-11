package main

import (
	"html/template"
	"log"
	"net/http"
)

var indexTmpl = template.Must(template.New("index.html").Parse(`<html>
  <head>
    <style>
form  { display: table;      }
p     { display: table-row;  }
label { display: table-cell; }
input { display: table-cell; }
    </style>
  </head>
  <body>
    <form action="/login" method="post">
      <p>
        <label> Authenticate for: </label>
        <input type="text" name="cross_client" placeholder="list of client-ids">
      </p>
      <p>
        <label>Extra scopes: </label>
        <input type="text" name="extra_scopes" placeholder="list of scopes">
      </p>
      <p>
        <label>Connector ID: </label>
        <input type="text" name="connector_id" placeholder="connector id">
      </p>
      <p>
        <label>Request offline access: </label>
        <input type="checkbox" name="offline_access" value="yes" checked>
      </p>
      <p>
	    <input type="submit" value="Login">
      </p>
    </form>
  </body>
</html>`))

func renderIndex(w http.ResponseWriter) {
	renderTemplate(w, indexTmpl, nil)
}

type tokenTmplData struct {
	IDToken      string
	AccessToken  string
	RefreshToken string
	RedirectURL  string
	Claims       string
}

var tokenTmpl = template.Must(template.New("token.html").Parse(`<html>
  <head>
    <style>
/* make pre wrap */
pre {
 white-space: pre-wrap;       /* css-3 */
 white-space: -moz-pre-wrap;  /* Mozilla, since 1999 */
 white-space: -pre-wrap;      /* Opera 4-6 */
 white-space: -o-pre-wrap;    /* Opera 7 */
 word-wrap: break-word;       /* Internet Explorer 5.5+ */
}
    </style>
  </head>
  <body>
    <p> ID Token: <pre><code>{{ .IDToken }}</code></pre></p>
    <p> Access Token: <pre><code>{{ .AccessToken }}</code></pre></p>
    <p> Claims: <pre><code>{{ .Claims }}</code></pre></p>
	{{ if .RefreshToken }}
    <p> Refresh Token: <pre><code>{{ .RefreshToken }}</code></pre></p>
	<form action="{{ .RedirectURL }}" method="post">
	  <input type="hidden" name="refresh_token" value="{{ .RefreshToken }}">
	  <input type="submit" value="Redeem refresh token">
    </form>
	{{ end }}
  </body>
</html>
`))

func renderToken(w http.ResponseWriter, redirectURL, idToken, accessToken, refreshToken, claims string) {
	renderTemplate(w, tokenTmpl, tokenTmplData{
		IDToken:      idToken,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		RedirectURL:  redirectURL,
		Claims:       claims,
	})
}

func renderTemplate(w http.ResponseWriter, tmpl *template.Template, data interface{}) {
	err := tmpl.Execute(w, data)
	if err == nil {
		return
	}

	switch err := err.(type) {
	case *template.Error:
		// An ExecError guarantees that Execute has not written to the underlying reader.
		log.Printf("Error rendering template %s: %s", tmpl.Name(), err)

		// TODO(ericchiang): replace with better internal server error.
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	default:
		// An error with the underlying write, such as the connection being
		// dropped. Ignore for now.
	}
}
