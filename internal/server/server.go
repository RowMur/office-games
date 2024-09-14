package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/RowMur/office-games/internal/database"
	"github.com/RowMur/office-games/internal/views"
	"gorm.io/gorm"
)

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Run() {
	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.Handle("/", authMiddleware(http.HandlerFunc(pageHandler)))
	http.Handle("/play", authMiddleware(http.HandlerFunc(playHandler)))
	http.Handle("/sign-in", http.HandlerFunc(signInHandler))
	http.Handle("/create-account", http.HandlerFunc(createAccountHandler))
	http.Handle("/me", authMiddleware(http.HandlerFunc(mePageHandler)))
	http.Handle("/sign-out", http.HandlerFunc(signOutHandler))
	http.Handle("/create-office", authMiddleware(http.HandlerFunc(createOfficeHandler)))
	http.Handle("/join-office", authMiddleware(http.HandlerFunc(joinOfficeHandler)))
	http.Handle("/office/", authMiddleware(http.HandlerFunc(officeHandler)))
	fmt.Println("Server is running on port 8080")
	http.ListenAndServe(":8080", nil)
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userOffices, err := user.GetOffices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userHasOffices := len(userOffices) > 0

	mainPageContent := views.MainPage(*user, userHasOffices, userOffices)
	views.Page(mainPageContent).Render(context.Background(), w)
}

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
		signInPageContent := views.SignInPage()
		views.Page(signInPageContent).Render(context.Background(), w)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	username := r.Form.Get("username")
	if username == "" {
		errs := views.FormErrors{"username": "Username is required"}
		views.SignInForm(views.FormData{}, errs).Render(context.Background(), w)
		return
	}

	user, err := database.GetUser(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if user == nil {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"username": "User not found"}
		views.SignInForm(data, errs).Render(context.Background(), w)
		return
	}

	cookie := fmt.Sprintf("auth=%s", username)
	w.Header().Set("Set-Cookie", cookie)
	w.Header().Set("HX-Redirect", "/")
}

func createAccountHandler(w http.ResponseWriter, r *http.Request) {
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
		createAccountPageContent := views.CreateAccountPage()
		views.Page(createAccountPageContent).Render(context.Background(), w)
		return
	}

	r.ParseForm()
	username := r.Form.Get("username")
	if username == "" {
		errs := views.FormErrors{"username": "Username is required"}
		views.SignInForm(views.FormData{}, errs).Render(context.Background(), w)
		return
	}

	user, err := database.GetUser(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if user != nil {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"username": "Username is taken"}
		views.SignInForm(data, errs).Render(context.Background(), w)
		return
	}

	_, err = database.CreateUser(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cookie := fmt.Sprintf("auth=%s", username)
	w.Header().Set("Set-Cookie", cookie)
	w.Header().Set("HX-Redirect", "/")
}

func playHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := userFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	opponentName := r.Form.Get("opponent")
	if opponentName == "" {
		http.Error(w, "Opponent name is required", http.StatusBadRequest)
		return
	}

	officeCode := r.URL.Query().Get("office")
	if officeCode == "" {
		http.Error(w, "Office code is required", http.StatusBadRequest)
		return
	}
	office, err := database.GetOfficeByCode(officeCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	opponent, err := office.FindPlayer(opponentName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if opponent == nil {
		http.Error(w, "Opponent not found", http.StatusNotFound)
		return
	}

	player, err := office.FindPlayer(user.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if player == nil {
		http.Error(w, "Player not found", http.StatusNotFound)
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

	database.GetDB().Save(&player)
	database.GetDB().Save(&opponent)

	players, err := office.GetPlayers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	views.OfficeRankings(players).Render(context.Background(), w)
}

func mePageHandler(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	mePageContent := views.MePage(*user)
	views.Page(mePageContent).Render(context.Background(), w)
}

func signOutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Set-Cookie", "auth=; Max-Age=0")
	http.Redirect(w, r, "/sign-in", http.StatusFound)
}

func createOfficeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := userFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	officeName := r.Form.Get("office")
	if officeName == "" {
		errs := views.FormErrors{"office": "Office name is required"}
		views.CreateOfficeForm(views.FormData{}, errs).Render(context.Background(), w)
		return
	}

	user.CreateOffice(officeName)
	w.Header().Set("HX-Redirect", "/")
}

func officeHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/")

	if path == "office" || path == "office/" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	officeCodeFromPath := strings.TrimPrefix(path, "office/")
	office, err := database.GetOfficeByCode(officeCodeFromPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if office == nil {
		http.Error(w, "Office not found", http.StatusNotFound)
		return
	}

	user := userFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	players, err := office.GetPlayers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	officePageContent := views.OfficePage(*office, players, *user)
	views.Page(officePageContent).Render(context.Background(), w)
}

func joinOfficeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := userFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	officeCode := r.Form.Get("office")
	if officeCode == "" {
		errs := views.FormErrors{"office": "Office code is required"}
		views.JoinOfficeForm(views.FormData{}, errs).Render(context.Background(), w)
		return
	}

	office, err := database.GetOfficeByCode(officeCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if office == nil {
		data := views.FormData{"office": officeCode}
		errs := views.FormErrors{"office": "Office not found"}
		views.JoinOfficeForm(data, errs).Render(context.Background(), w)
		return
	}

	player, err := office.FindPlayer(user.Username)
	if err != nil && err != gorm.ErrRecordNotFound {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if player != nil {
		data := views.FormData{"office": officeCode}
		errs := views.FormErrors{"office": "Already in the office"}
		views.JoinOfficeForm(data, errs).Render(context.Background(), w)
		return
	}

	_, err = office.AddPlayer(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Redirect", fmt.Sprintf("/office/%s", office.Code))
}
