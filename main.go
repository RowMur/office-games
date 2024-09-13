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

func (o *Office) FindPlayer(name string) *Player {
	for _, p := range o.Players {
		if p.Name == name {
			return p
		}
	}

	return nil
}

func newOffice(name string) *Office {
	office := &Office{
		Name: name,
		Players: []*Player{
			newPlayer("Rowan"),
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
	http.Handle("/", authMiddleware(http.HandlerFunc(pageHandler)))
	http.Handle("/play", authMiddleware(http.HandlerFunc(playHandler)))
	http.Handle("/sign-in", http.HandlerFunc(signInHandler))
	fmt.Println("Server is running on port 8080")
	http.ListenAndServe(":8080", nil)
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	mainPageContent := mainPage(*office, *user)
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

func playHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	player := userFromContext(r.Context())
	if player == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	opponentName := r.Form.Get("opponent")
	if opponentName == "" {
		http.Error(w, "Opponent name is required", http.StatusBadRequest)
		return
	}

	opponent := office.FindPlayer(opponentName)
	if opponent == nil {
		http.Error(w, "Opponent not found", http.StatusNotFound)
		return
	}

	didWin := r.Form.Get("win") == "on"
	if didWin {
		player.Points += 10
		opponent.Points -= 10
	} else {
		player.Points -= 10
		opponent.Points += 10
	}

	office.SortPlayers()
	w.Header().Set("HX-Redirect", "/")
}

type contextKey string

const userContextKey = contextKey("user")

func userFromContext(ctx context.Context) *Player {
	user, ok := ctx.Value(userContextKey).(*Player)
	if !ok {
		return nil
	}
	return user
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCookie, err := r.Cookie("auth")
		if err != nil {
			if err != http.ErrNoCookie {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/sign-in", http.StatusFound)
			return
		}

		player := office.FindPlayer(authCookie.Value)
		if player == nil {
			http.Redirect(w, r, "/sign-in", http.StatusFound)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, player)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
