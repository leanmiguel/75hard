package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"leanmiguel/75hard/pkg/forms"
	"leanmiguel/75hard/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/grsmv/goweek"
)

type ChallengeRequest struct {
	Challenge int  `json:"challenge"`
	Checked   bool `json:"checked"`
}

type ChallengeResponse struct {
	Checked bool `json:"checked"`
}

func (app *application) updateChallenge(w http.ResponseWriter, r *http.Request) {
	var cReq ChallengeRequest

	userId := app.session.GetInt(r.Context(), "authenticatedUserID")

	err := json.NewDecoder(r.Body).Decode(&cReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = app.db.UpdateUserChallenge(context.TODO(), userId, time.Now(), cReq.Challenge, cReq.Checked)

	if err != nil {
		panic("UHO HOHO")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChallengeResponse{Checked: cReq.Checked})

}

func (app *application) homePage(w http.ResponseWriter, r *http.Request) {

	//TODO: this can also take a URL PARAM here
	//TODO: client error if bad input, must be a date

	userId := app.session.GetInt(r.Context(), "authenticatedUserID")

	settings, err := app.db.GetSettingsByUser(context.TODO(), userId)

	if err != nil {
		panic(err)
	}

	//TODO: if no date is found, then this is a not found error
	challenges, err := app.db.GetUserChallengesByDay(context.TODO(), userId, time.Now())

	if err != nil {
		panic(err)
	}

	//TODO: convert this to data
	type Data struct {
		Settings   web.Settings
		Challenges web.Challenges
	}

	app.render(w, r, "home.page.tmpl", &templateData{
		Settings:   *settings,
		Challenges: *challenges,
	})

}

func (app *application) communityPage(w http.ResponseWriter, r *http.Request) {

	//TODO: params with wrong formatting is client error

	//TODO: change now to be default to today, or take from the url
	challenges, err := app.db.GetCommunityChallengesByDay(r.Context(), time.Now())

	//TODO: this should be server error here if something goes wrong here
	if err != nil {
		panic(err)
	}

	var challengesWithUsername []web.ChallengeWithUsername

	for _, v := range *challenges {
		challUser, err := app.db.GetChallengeUsernameStruct(v)

		//TODO: this should also be a server error here
		if err != nil {
			app.serverError(w, err)
		}

		challengesWithUsername = append(challengesWithUsername, challUser)
	}

	//TODO: refactor date to be the passed in date
	//also update render
	app.render(w, r, "community.page.tmpl", &templateData{
		CommunityDate:       time.Now().Format("January 02"),
		CommunityChallenges: challengesWithUsername,
	})

}

func (app *application) historyPage(w http.ResponseWriter, r *http.Request) {

	//TODO: params with wrong formatting is client error

	// TODO: use a url param here, and then determine which week.
	year, week := time.Now().ISOWeek()

	// TODO: make a function to calculate the week.
	gweek, err := goweek.NewWeek(year, week)

	//TODO: this a server error
	if err != nil {
		panic(err)
	}

	days := gweek.Days

	userId := app.session.GetInt(r.Context(), "authenticatedUserID")

	challenges, err := app.db.GetUserChallengesByWeek(r.Context(), userId, days)

	var formattedDateChallenges []web.ChallengeWithDateString

	for _, v := range *challenges {
		formattedChallenge := web.ChallengeWithDateString{
			UserId: v.UserId,
			First:  v.First,
			Second: v.Second,
			Third:  v.Third,
			Fourth: v.Fourth,
			Fifth:  v.Fifth,
			Month:  v.Date.Format("09"),
			Day:    strconv.Itoa(v.Date.Day()),
		}
		formattedDateChallenges = append(formattedDateChallenges, formattedChallenge)
	}

	if err != nil {
		panic(err)
	}

	// reverse the array
	for i, j := 0, len(formattedDateChallenges)-1; i < j; i, j = i+1, j-1 {
		formattedDateChallenges[i], formattedDateChallenges[j] = formattedDateChallenges[j], formattedDateChallenges[i]
	}

	app.render(w, r, "history.page.tmpl", &templateData{
		HistoryWeekChallenges: formattedDateChallenges,
	})

}

func (app *application) settingsPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "settings.page.tmpl", nil)

}

func (app *application) loginPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "login.page.tmpl", &templateData{
		Form: forms.New(nil),
	})

}

func (app *application) signupPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "signup.page.tmpl", &templateData{
		Form: forms.New(nil),
	})

}

func (app *application) handleLogin(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()

	if err != nil {
		panic(err)
	}

	form := forms.New(r.PostForm)

	form.Required("username", "password")

	if !form.Valid() {
		app.render(w, r, "login.page.tmpl", &templateData{
			Form: form,
		})
		return
	}
	id, err := app.db.AuthenticateUser(form.Values.Get("username"), form.Values.Get("password"))

	if err != nil {
		if errors.Is(err, web.ErrInvalidCredentials) {
			form.Errors.Add("generic", "Email or Password is incorrect")
			app.render(w, r, "login.page.tmpl", &templateData{Form: form})
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.session.Put(r.Context(), "authenticatedUserID", id)

	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func (app *application) registerUser(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)

	form.MinLength("password", 3)
	form.Required("username", "password")
	form.MaxLength("username", 15)

	if !form.Valid() {
		app.render(w, r, "signup.page.tmpl", &templateData{
			Form: form,
		})
		return
	}

	err = app.db.CreateUser(r.Context(), form.Get("username"), form.Get("password"))

	if err != nil {
		if errors.Is(err, web.ErrDuplicateUser) {
			form.Errors.Add("username", "username is already in use")
			app.render(w, r, "signup.page.tmpl", &templateData{
				Form: form,
			})
			return
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.session.Put(r.Context(), "flash", "You've been signed up, go ahead and login!")

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (app *application) handleLogout(w http.ResponseWriter, r *http.Request) {
	app.session.Remove(r.Context(), "authenticatedUserID")
	app.session.Put(r.Context(), "flash", "You've been logged out successfully!")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) routes() *chi.Mux {

	r := chi.NewMux()

	r.Use(app.session.LoadAndSave)

	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	r.Group((func(r chi.Router) {
		r.Use(app.requireAuthentication)
		r.Get("/", app.homePage)
		r.Get("/community", app.communityPage)
		r.Get("/history", app.historyPage)
		r.Get("/settings", app.settingsPage)
		r.Post("/challenge/*", app.updateChallenge)
	}))

	r.Get("/login", app.loginPage)
	r.Get("/signup", app.signupPage)
	r.Post("/login", app.handleLogin)
	r.Post("/signup", app.registerUser)
	r.Post("/logout", app.handleLogout)

	fs := http.FileServer(http.Dir("./ui/static"))
	r.Handle("/static/*", http.StripPrefix("/static", fs))

	return r
}
