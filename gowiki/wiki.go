package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
)

type Page struct {
	Title string
	Body  []byte
}

var templates = template.Must(template.ParseFiles("edit.html", "view.html"))
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return os.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	log.Printf("Fetching view page of title: %v\n", title)
	p, err := loadPage(title)
	if err != nil {
		log.Printf("Not found, redirecting...\n")

		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	log.Printf("Loading edit page of title: %v\n", title)
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	log.Printf("Saving %v to page of title %v\n", body, title)
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Entering decorated function...")
		log.Printf("URL Path: %v\n", r.URL.Path)
		m := validPath.FindStringSubmatch(r.URL.Path)
		log.Printf("String Sub Matches: %v", m)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		m_1 := m[1]
		log.Printf("m_1: %v", m_1)
		m_2 := m[2]
		log.Printf("m_2: %v", m_2)
		fn(w, r, m_2)
	}
}

func main() {
	log.SetPrefix("web: ")
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
