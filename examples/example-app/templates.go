package main

import (
	"html/template"
	"log"
	"net/http"
)

const css = `
	body {
		font-family: Arial, sans-serif;
		background-color: #f2f2f2;
		margin: 0;
	}

    .header {
        text-align: center;
        margin-bottom: 20px;
    }

	.dex {
		font-size: 2em;
		font-weight: bold;
		color: #3F9FD8; /* Main color */
	}

	.example-app {
		font-size: 1em;
		color: #EF4B5C; /* Secondary color */
	}

	.form-instructions {
		text-align: center;
		margin-bottom: 15px;
		font-size: 1em;
		color: #555;
	}

	hr {
		border: none;
		border-top: 1px solid #ccc;
		margin-top: 10px;
		margin-bottom: 20px;
	}

	label {
		flex: 1;
		font-weight: bold;
		color: #333;
	}

	p {
		margin-bottom: 15px;
		display: flex;
		align-items: center;
	}

	input[type="text"] {
		flex: 2;
		padding: 8px;
		border: 1px solid #ccc;
		border-radius: 4px;
		outline: none;
	}

	input[type="checkbox"] {
		margin-left: 10px;
		transform: scale(1.2);
	}

	.back-button {
		display: inline-block;
		padding: 8px 16px;
		background-color: #EF4B5C; /* Secondary color */
		color: white;
		border: none;
		border-radius: 4px;
		cursor: pointer;
		font-size: 12px;
		text-decoration: none;
		transition: background-color 0.3s ease, transform 0.2s ease;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
		position: fixed;
		right: 20px;
		bottom: 20px;
	}

	.back-button:hover {
		background-color: #C43B4B; /* Darker shade of secondary color */
	}

	.token-block {
		background-color: #fff;
		padding: 10px 15px;
		border-radius: 8px;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
		margin-bottom: 15px;
		word-wrap: break-word;
		display: flex;
		flex-direction: column;
		gap: 5px;
		position: relative;
	}

	.token-title {
		font-weight: bold;
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.token-title a {
		font-size: 0.9em;
		text-decoration: none;
		color: #3F9FD8; /* Main color */
	}

	.token-title a:hover {
		text-decoration: underline;
	}

    .token-code {
		overflow-wrap: break-word;
		word-break: break-all;
		white-space: normal;
    }

	pre {
		white-space: pre-wrap;
		background-color: #f9f9f9;
		padding: 8px;
		border-radius: 4px;
		border: 1px solid #ddd;
		margin: 0;
		font-family: 'Courier New', Courier, monospace;
		overflow-x: auto;
		font-size: 0.9em;
		position: relative;
		margin-top: 5px;
	}

	pre .key {
		color: #c00;
	}

	pre .string {
		color: #080;
	}

	pre .number {
		color: #00f;
	}
`

var indexTmpl = template.Must(template.New("index.html").Parse(`<html>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Example App - Login</title>
    <style>
` + css + `
        body {
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            flex-direction: column;
        }

		form {
			background-color: #fff;
			padding: 20px;
			border-radius: 8px;
			box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
			width: 100%;
			max-width: 400px;
		}

		input[type="submit"] {
			width: 100%;
			padding: 10px;
			background-color: #3F9FD8; /* Main color */
			color: white;
			border: none;
			border-radius: 4px;
			cursor: pointer;
			font-size: 16px;
		}

		input[type="submit"]:hover {
			background-color: #357FAA; /* Darker shade of main color */
		}
    </style>
</head>
<body>
    <div class="header">
        <div class="dex">Dex</div>
        <div class="example-app">Example App</div>
    </div>
    <form action="/login" method="post">
        <div class="form-instructions">
            If needed, customize your login settings below, then click <strong>Login</strong> to proceed.
        </div>
        <hr/>
        <p>
            <label for="cross_client">Authenticate for:</label>
            <input type="text" id="cross_client" name="cross_client" placeholder="list of client-ids">
        </p>
        <p>
            <label for="extra_scopes">Extra scopes:</label>
            <input type="text" id="extra_scopes" name="extra_scopes" placeholder="list of scopes">
        </p>
        <p>
            <label for="connector_id">Connector ID:</label>
            <input type="text" id="connector_id" name="connector_id" placeholder="connector id">
        </p>
        <p>
            <label for="offline_access">Request offline access:</label>
            <input type="checkbox" id="offline_access" name="offline_access" value="yes" checked>
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
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Tokens</title>
    <style>
` + css + `
        body {
            color: #333;
            margin: 0;
            padding: 20px;
            position: relative;
        }

        input[type="submit"] {
            margin-top: 10px;
            padding: 8px 16px;
            background-color: #3F9FD8; /* Main color */
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            transition: background-color 0.3s ease;
        }

        input[type="submit"]:hover {
            background-color: #357FAA; /* Darker shade of main color */
        }
    </style>
</head>
<body>
    {{ if .IDToken }}
    <div class="token-block">
        <div class="token-title">
            ID Token:
            <a href="#" onclick="window.open('https://jwt.io/#debugger-io?token=' + encodeURIComponent('{{ .IDToken }}'), '_blank')">Decode on jwt.io</a>
        </div>
        <pre><code class="token-code">{{ .IDToken }}</code></pre>
    </div>
    {{ end }}

    {{ if .AccessToken }}
    <div class="token-block">
        <div class="token-title">
            Access Token:
            <a href="#" onclick="window.open('https://jwt.io/#debugger-io?token=' + encodeURIComponent('{{ .AccessToken }}'), '_blank')">Decode on jwt.io</a>
        </div>
        <pre><code class="token-code">{{ .AccessToken }}</code></pre>
    </div>
    {{ end }}

    {{ if .Claims }}
    <div class="token-block">
        <div class="token-title">Claims:</div>
        <pre><code id="claims">{{ .Claims }}</code></pre>
    </div>
    {{ end }}

    {{ if .RefreshToken }}
    <div class="token-block">
        <div class="token-title">Refresh Token:</div>
        <pre><code class="token-code">{{ .RefreshToken }}</code></pre>
        <form action="{{ .RedirectURL }}" method="post">
            <input type="hidden" name="refresh_token" value="{{ .RefreshToken }}">
            <input type="submit" value="Redeem refresh token">
        </form>
    </div>
    {{ end }}

    <a href="/" class="back-button">Back to Home</a>

    <script>
        // Simple JSON syntax highlighter
        document.addEventListener("DOMContentLoaded", function() {
            const claimsElement = document.getElementById("claims");
            if (claimsElement) {
                try {
                    const json = JSON.parse(claimsElement.textContent);
                    claimsElement.innerHTML = syntaxHighlight(json);
                } catch (e) {
                    console.error("Invalid JSON in claims:", e);
                }
            }
        });

        function syntaxHighlight(json) {
            if (typeof json != 'string') {
                json = JSON.stringify(json, undefined, 2);
            }
            json = json.replace(/&/g, '&amp;').replace(/</g, '<').replace(/>/g, '>');
            return json.replace(/("(\\u[\da-fA-F]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|\b\d+\b)/g, function (match) {
                let cls = 'number';
                if (/^"/.test(match)) {
                    if (/:$/.test(match)) {
                        cls = 'key';
                    } else {
                        cls = 'string';
                    }
                } else if (/true|false/.test(match)) {
                    cls = 'boolean';
                } else if (/null/.test(match)) {
                    cls = 'null';
                }
                return '<span class="' + cls + '">' + match + '</span>';
            });
        }
    </script>
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
