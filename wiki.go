// Go wiki from https://golang.org/doc/articles/wiki/ using httpmux.
package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"

	"golang.org/x/net/context"

	"github.com/go-web/httplog"
	"github.com/go-web/httpmux"
)

var (
	templates = template.Must(template.ParseFiles("edit.html", "view.html"))
	validPath = regexp.MustCompile("^[a-zA-Z0-9]+$")
)

func main() {
	mux := httpmux.New()
	logger := log.New(os.Stderr, "[wiki] ", 0)
	mux.Use(httplog.ApacheCombinedFormat(logger))
	mux.Use(titleValidation)
	mux.GET("/view/:title", viewHandler)
	mux.GET("/edit/:title", editHandler)
	mux.POST("/save/:title", saveHandler)
	log.Println("Visit http://localhost:8080/view/hello")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		logger.Fatal(err)
	}
}

func titleValidation(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		title := httpmux.Params(r).ByName("title")
		if !validPath.MatchString(title) {
			http.NotFound(w, r)
			return
		}
		ctx := context.WithValue(httpmux.Context(r), "title", title)
		httpmux.SetContext(ctx, r)
		next(w, r)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	title := httpmux.Context(r).Value("title").(string)
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	title := httpmux.Context(r).Value("title").(string)
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	title := httpmux.Context(r).Value("title").(string)
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

type Page struct {
	Title string
	Body  []byte
}

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
