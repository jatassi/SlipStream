package plextv

import (
	"html/template"
	"net/http"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
)

func newAuth(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
	s := stateOf(env.Shared)
	return fake.HTTP(env, fake.Routes{
		"page":    {Pattern: "GET /auth", Handler: s.page},
		"approve": {Pattern: "POST /auth/approve", Handler: s.approvePage},
		"decline": {Pattern: "POST /auth/decline", Handler: declinePage},
	}), nil
}

// pageHTML is the sign-in page. The PIN's code and forwardUrl travel in
// the URL fragment, as on app.plex.tv, which a browser never sends, so
// the script copies them into the forms. With more than one account in
// the world it asks which signs in, the Admin's first.
var pageHTML = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Sign in to Plex</title></head>
<body>
<h1>Sign in to Plex</h1>
<p><span id="product">An app</span> wants to use your Plex account. Code <code id="code"></code>.</p>
<form method="post" action="/auth/approve"><input type="hidden" name="code"><input type="hidden" name="forwardUrl">
{{if gt (len .) 1}}<label>Plex account <select name="account">{{range .}}<option>{{.}}</option>{{end}}</select></label>{{end}}
<button type="submit">Allow</button></form>
<form method="post" action="/auth/decline"><input type="hidden" name="forwardUrl"><button type="submit">Decline</button></form>
<script>
const params = new URLSearchParams(location.hash.replace(/^#!?\??/, ""));
document.getElementById("code").textContent = params.get("code") ?? "";
if (params.get("context[device][product]")) document.getElementById("product").textContent = params.get("context[device][product]");
for (const input of document.querySelectorAll("input[name=code]")) input.value = params.get("code") ?? "";
for (const input of document.querySelectorAll("input[name=forwardUrl]")) input.value = params.get("forwardUrl") ?? "";
</script>
</body>
</html>
`))

var doneHTML = template.Must(template.New("done").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>Sign in to Plex</title></head><body><p>{{.}}</p></body></html>
`))

func (s *state) page(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	names := make([]string, len(s.accounts))
	for i, a := range s.accounts {
		names[i] = a.Username
	}
	s.mu.Unlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pageHTML.Execute(w, names)
}

func done(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = doneHTML.Execute(w, message)
}

// forward opens forwardUrl again exactly as the app gave it, with no
// query string added, as the recorded run saw.
func forward(w http.ResponseWriter, r *http.Request, message string) {
	if to := r.PostFormValue("forwardUrl"); to != "" {
		http.Redirect(w, r, to, http.StatusSeeOther) //nolint:gosec // plex.tv returns to whatever forwardUrl the app named, and so does the fake
		return
	}
	done(w, http.StatusOK, message)
}

func (s *state) approvePage(w http.ResponseWriter, r *http.Request) {
	code := r.PostFormValue("code")
	if code == "" {
		done(w, http.StatusBadRequest, "This page has no code. Start the sign-in again in the app.")
		return
	}
	if _, err := s.approve(code, r.PostFormValue("account")); err != nil {
		done(w, http.StatusNotFound, "This code has expired. Start the sign-in again in the app.")
		return
	}
	forward(w, r, "You're signed in to Plex. Close this window to go back to the app.")
}

func declinePage(w http.ResponseWriter, r *http.Request) {
	forward(w, r, "The app isn't signed in to Plex. Close this window to go back to the app.")
}
