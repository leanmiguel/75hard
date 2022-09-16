package main

import (
	"html/template"
	"leanmiguel/75hard/pkg/forms"
	"leanmiguel/75hard/web"
	"net/http"

	"path/filepath"
	"time"
)

// Define a templateData type to act as the holding structure for
// any dynamic data that we want to pass to our HTML templates.
// At the moment it only contains one field, but we'll add more
// to it as the build progresses.
type templateData struct {
	CurrentYear           int
	Form                  *forms.Form
	Flash                 string
	IsAuthenticated       bool
	Settings              web.Settings
	Challenges            web.Challenges
	CommunityDate         string
	CommunityChallenges   []web.ChallengeWithUsername
	HistoryWeekChallenges []web.ChallengeWithDateString
}

func newTemplateCache(dir string) (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := filepath.Glob(filepath.Join(dir, "*.page.tmpl"))
	if err != nil {
		return nil, err
	}

	// Loop through the pages one-by-one.
	for _, page := range pages {

		name := filepath.Base(page)

		ts, err := template.New(name).ParseFiles(page)
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseGlob(filepath.Join(dir, "*.layout.tmpl"))
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseGlob(filepath.Join(dir, "*.partial.tmpl"))
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}

func humanDate(t time.Time) string {
	return t.Format("02 Jan 2006 at 15:04")
}

func (app *application) addDefaultData(td *templateData, r *http.Request) *templateData {
	if td == nil {
		td = &templateData{}
	}

	td.Flash = app.session.PopString(r.Context(), "flash")
	td.CurrentYear = time.Now().Year()

	td.IsAuthenticated = app.isAuthenticated(r)

	return td
}
