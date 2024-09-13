package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
)

type Player struct {
	Name   string
	Points int
}

func newPlayer(name string) *Player {
	return &Player{
		Name:   name,
		Points: rand.Intn(1000),
	}
}

type Office struct {
	Name    string
	Players []*Player
}

func (o *Office) SortPlayers() {
	sort.Slice(o.Players, func(i, j int) bool {
		return o.Players[i].Points > o.Players[j].Points
	})
}

func (o *Office) AddPlayer(name string) {
	o.Players = append(o.Players, newPlayer(name))
	o.SortPlayers()
}

func newOffice(name string) *Office {
	office := &Office{
		Name: name,
		Players: []*Player{
			newPlayer("John"),
			newPlayer("Jane"),
			newPlayer("Jack"),
			newPlayer("Jill"),
		},
	}

	office.SortPlayers()
	return office
}

var office = newOffice("Office")

func main() {
	pageHandlerFunc := http.HandlerFunc(pageHandler)
	http.Handle("/", authMiddleware(pageHandlerFunc))
	http.Handle("/sign-in", http.HandlerFunc(signInHandler))
	fmt.Println("Server is running on port 8080")
	http.ListenAndServe(":8080", nil)
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	mainPageContent := mainPage(*office)
	page(mainPageContent).Render(context.Background(), w)
}

type FormErrors map[string]string

func signInHandler(w http.ResponseWriter, r *http.Request) {
	authCookie, err := r.Cookie("auth")
	if err != nil && err != http.ErrNoCookie {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if authCookie != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == http.MethodGet {
		signInPageContent := signInPage()
		page(signInPageContent).Render(context.Background(), w)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	name := r.Form.Get("name")
	if name == "" {
		errs := FormErrors{"name": "Name is required"}
		signInForm(errs).Render(context.Background(), w)
		return
	}

	office.AddPlayer(name)
	cookie := fmt.Sprintf("auth=%s", name)
	w.Header().Set("Set-Cookie", cookie)
	w.Header().Set("HX-Redirect", "/")
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie("auth")
		if err != nil {
			if err != http.ErrNoCookie {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else {
				http.Redirect(w, r, "/sign-in", http.StatusFound)
			}
		}

		next.ServeHTTP(w, r)
	})
}
