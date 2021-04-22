package auth

import (
	"errors"
	"html/template"
	"log"
	"net/http"
)

type tokenTmplData struct {
	IDToken        string
	RefreshToken   string
	RedirectURL    string
	Claims         string
	SuccessMessage string
	Debug          bool
}

var tokenTmpl = template.Must(template.New("token.html").Parse(`<html>
  <head>
    <style>
html {
  font-family: sans-serif; /* 1 */
  -ms-text-size-adjust: 100%; /* 2 */
  -webkit-text-size-adjust: 100%; /* 2 */
}
body {
  margin: 50;
  background: #F0F0F0;
}
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
		<h3>Authentication successful</h3>
		<h4>{{ .SuccessMessage }}</h4>
	{{ if .Debug }}
		<p> Token: <pre><code>{{ .IDToken }}</code></pre></p>
    <p> Claims: <pre><code>{{ .Claims }}</code></pre></p>
		{{ if .RefreshToken }}
    <p> Refresh Token: <pre><code>{{ .RefreshToken }}</code></pre></p>
		{{ end }}
	{{ end }}
  </body>
</html>
`))

// renders response page in browser which is displayed to the user at the end of the oidc-flow
func renderToken(w http.ResponseWriter, idToken, refreshToken string, claims []byte, successMessage string, debug bool) {
	renderTemplate(w, tokenTmpl, tokenTmplData{
		IDToken:        idToken,
		RefreshToken:   refreshToken,
		Claims:         string(claims),
		SuccessMessage: successMessage,
		Debug:          debug,
	})
}

func renderTemplate(w http.ResponseWriter, tmpl *template.Template, data interface{}) {
	err := tmpl.Execute(w, data)
	if err == nil {
		return
	}

	var templateErr *template.Error
	if errors.As(err, &templateErr) {
		// An ExecError guarantees that Execute has not written to the underlying reader.
		log.Printf("Error rendering template %s: %s", tmpl.Name(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	} else {
		log.Printf("Error rendering template %s: %s", tmpl.Name(), err)
	}
}
