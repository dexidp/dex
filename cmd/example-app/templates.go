package main

import (
	"html/template"
	"log"
	"net/http"
)

var indexTmpl = template.Must(template.New("index.html").Parse(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <!-- The above 3 meta tags *must* come first in the head; any other head content must come *after* these tags -->
    <title>KubeAuth Login</title>

    <!-- Bootstrap -->
		<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css">
		<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap-theme.min.css">
		<script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js"></script>

    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/bootstrap-social/5.1.1/bootstrap-social.min.css">
    <!-- HTML5 shim and Respond.js for IE8 support of HTML5 elements and media queries -->
    <!-- WARNING: Respond.js doesn't work if you view the page via file:// -->
    <!--[if lt IE 9]>
      <script src="https://oss.maxcdn.com/html5shiv/3.7.3/html5shiv.min.js"></script>
      <script src="https://oss.maxcdn.com/respond/1.4.2/respond.min.js"></script>
    <![endif]-->
    <style>
      body {
        padding: 40px;
      }
    </style>
  </head>
  <body>
    <div class="row">
      <div class="col-xs-12 col-sm-4 col-md-3">
    		<form action="/login" method="post">
      		<div style="display:none;">
      			 <p>
      				 Authenticate for:<input type="text" name="cross_client" placeholder="list of client-ids">
      			 </p>
      			 <p>
      				 Extra scopes:<input type="text" name="extra_scopes" placeholder="list of scopes">
      			 </p>
        		 <p>
      			   Request offline access:<input type="checkbox" name="offline_access" value="yes" checked>
      			 </p>
      		 </div>
    			 <!-- <input type="submit" value="Login"> -->
           <button type="submit" class="btn btn-block btn-social btn-github">
             <span class="fa fa-github"></span>
             Sign in with GitHub
           </button>
    		</form>
      </div>
    </div>
  </body>
</html>`))

func renderIndex(w http.ResponseWriter) {
	renderTemplate(w, indexTmpl, nil)
}

type tokenTmplData struct {
	IDToken      string
	RefreshToken string
	RedirectURL  string
	Claims       string
}

var tokenTmpl = template.Must(template.New("token.html").Parse(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <!-- The above 3 meta tags *must* come first in the head; any other head content must come *after* these tags -->
    <title>KubeAuth Login</title>

    <!-- Bootstrap -->
		<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css">
		<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap-theme.min.css">
		<script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js"></script>

    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/bootstrap-social/5.1.1/bootstrap-social.min.css">
    <!-- HTML5 shim and Respond.js for IE8 support of HTML5 elements and media queries -->
    <!-- WARNING: Respond.js doesn't work if you view the page via file:// -->
    <!--[if lt IE 9]>
      <script src="https://oss.maxcdn.com/html5shiv/3.7.3/html5shiv.min.js"></script>
      <script src="https://oss.maxcdn.com/respond/1.4.2/respond.min.js"></script>
    <![endif]-->
    <style>
      body {
        padding: 40px;
      }
			.pre-y-scrollable {
        word-wrap: break-word;
        white-space: pre-wrap;
      }
    </style>
  </head>
  <body>
    <div class="row">
      <div class="col-xs-12 col-sm-6 col-md-6">
        <div class="panel panel-primary">
          <div class="panel-heading">ID Token</div>
          <div class="panel-body">
            <pre class="pre-y-scrollable">{{ .IDToken }}</pre>
          </div>
        </div>
        <div class="panel panel-info">
          <div class="panel-heading">Claims</div>
          <div class="panel-body">
            <pre>{{ .Claims }}</pre>
          </div>
        </div>
        <div class="panel panel-success">
          <div class="panel-heading">Refresh Token</div>
          <div class="panel-body">
            {{ if .RefreshToken }}
              <pre>{{ .RefreshToken }}</pre>
              <form action="{{ .RedirectURL }}" method="post">
            	  <input type="hidden" name="refresh_token" value="{{ .RefreshToken }}">
                <button type="submit" class="btn btn-success">
                  <span class="fa fa-refresh"></span>
                  Redeem New Tokens
                </button>
              </form>
            {{ end }}
          </div>
        </div>
      </div>
    </div>
  </body>
</html>`))

func renderToken(w http.ResponseWriter, redirectURL, idToken, refreshToken string, claims []byte) {
	renderTemplate(w, tokenTmpl, tokenTmplData{
		IDToken:      idToken,
		RefreshToken: refreshToken,
		RedirectURL:  redirectURL,
		Claims:       string(claims),
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
