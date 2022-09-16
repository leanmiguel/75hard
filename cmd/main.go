package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"leanmiguel/75hard/web"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
)

type application struct {
	db            web.SQLDB
	session       scs.SessionManager
	errorLog      log.Logger
	infoLog       log.Logger
	templateCache map[string]*template.Template
}

const TASKS_TABLE = "Tasks"

func main() {

	port := os.Getenv("PORT")

	db := web.NewDB()
	defer db.Close()

	sessionManager := scs.New()
	sessionManager.Store = postgresstore.New(db)

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	templateCache, err := newTemplateCache("./ui/html/")

	if err != nil {
		errorLog.Fatal(err)
	}

	app := application{
		db:            web.SQLDB{DB: db},
		session:       *sessionManager,
		errorLog:      *errorLog,
		infoLog:       *infoLog,
		templateCache: templateCache,
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: app.routes(),
	}

	err = app.db.InitTables()
	if err != nil {
		panic(err)
	}

	fmt.Println("listening at :", port)
	server.ListenAndServe()

}
