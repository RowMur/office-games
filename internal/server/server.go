package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/views"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	userHasOffices := len(user.Offices) > 0

	mainPageContent := views.MainPage(*user, userHasOffices, user.Offices)
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
	password := r.Form.Get("password")
	if username == "" || password == "" {
		data := views.FormData{"username": username}
		errs := views.FormErrors{}
		if username == "" {
			errs["username"] = "Username is required"
		}
		if password == "" {
			errs["password"] = "Password is required"
		}
		views.SignInForm(data, errs).Render(context.Background(), w)
		return
	}

	user := &db.User{}
	err = db.GetDB().Where("username = ?", username).First(user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"username": "User not found"}
		views.SignInForm(data, errs).Render(context.Background(), w)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"password": "Invalid password"}
		views.SignInForm(data, errs).Render(context.Background(), w)
		return
	}

	token, err := generateToken(int(user.ID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cookie := fmt.Sprintf("auth=%s", token)
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
	email := r.Form.Get("email")
	password := r.Form.Get("password")
	confirm := r.Form.Get("confirm")
	if username == "" || email == "" || password == "" || confirm == "" {
		data := views.FormData{"username": username, "email": email, "password": password, "confirm": confirm}
		errs := views.FormErrors{}
		if username == "" {
			errs["username"] = "Username is required"
		}
		if email == "" {
			errs["email"] = "Email is required"
		}
		if password == "" {
			errs["password"] = "Password is required"
		}
		if confirm == "" {
			errs["confirm"] = "Confirm password is required"
		}
		views.CreateAccountForm(data, errs).Render(context.Background(), w)
		return
	}

	if password != confirm {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"confirm": "Passwords do not match"}
		views.CreateAccountForm(data, errs).Render(context.Background(), w)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user := &db.User{Username: username, Email: email, Password: string(hashedPassword)}
	err = db.GetDB().Create(user).Error
	if err != nil {
		postgresErr, ok := err.(*pgconn.PgError)
		if !ok {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		formData := views.FormData{"username": username, "email": email}

		// Check if the error is a unique constraint violation
		if postgresErr.SQLState() == "23505" {
			constaintArray := strings.Split(postgresErr.ConstraintName, "_")
			columnName := constaintArray[len(constaintArray)-1]

			if columnName == "username" {
				errs := views.FormErrors{"username": "Username is taken"}
				views.CreateAccountForm(formData, errs).Render(context.Background(), w)
				return
			}

			if columnName == "email" {
				errs := views.FormErrors{"email": "Email is taken"}
				views.CreateAccountForm(formData, errs).Render(context.Background(), w)
				return
			}
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := generateToken(int(user.ID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cookie := fmt.Sprintf("auth=%s", token)
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
	opponentID, err := strconv.Atoi(r.Form.Get("opponent"))
	if err != nil {
		http.Error(w, "Invalid opponent", http.StatusBadRequest)
		return
	}

	gameID, err := strconv.Atoi(r.URL.Query().Get("game"))
	if err != nil {
		http.Error(w, "Invalid game", http.StatusBadRequest)
		return
	}

	opponentIDUint := uint(opponentID)

	var winnerID uint
	var loserID uint
	if r.Form.Get("win") == "on" {
		winnerID = user.ID
		loserID = opponentIDUint
	} else {
		winnerID = opponentIDUint
		loserID = user.ID
	}

	match := &db.Match{
		GameID:   uint(gameID),
		WinnerID: winnerID,
		LoserID:  loserID,
	}

	dbc := db.GetDB()

	err = dbc.Create(match).Error
	if err != nil {
		if errors.Is(err, gorm.ErrForeignKeyViolated) {
			http.Error(w, "Invalid game or opponent", http.StatusBadRequest)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var rankings []db.Ranking
	err = dbc.Where("game_id = ?", gameID).Preload("User").Order("Points desc").Find(&rankings).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	game := &db.Game{}
	err = dbc.Where("id = ?", gameID).Preload("Matches").First(game).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userWinLosses := map[uint]views.WinLosses{}
	for _, match := range game.Matches {
		userWinLosses[match.WinnerID] = views.WinLosses{
			Wins:   userWinLosses[match.WinnerID].Wins + 1,
			Losses: userWinLosses[match.WinnerID].Losses,
		}

		userWinLosses[match.LoserID] = views.WinLosses{
			Wins:   userWinLosses[match.LoserID].Wins,
			Losses: userWinLosses[match.LoserID].Losses + 1,
		}
	}

	views.OfficeRankings(rankings, userWinLosses).Render(context.Background(), w)
}

func mePageHandler(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		mePageContent := views.MePage(*user, views.FormData{"email": user.Email, "username": user.Username}, nil)
		views.Page(mePageContent).Render(context.Background(), w)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	username := r.Form.Get("username")
	email := r.Form.Get("email")

	if username == "" || email == "" {
		data := views.FormData{"username": username, "email": email}
		errs := views.FormErrors{}
		if username == "" {
			errs["username"] = "Username is required"
		}
		if email == "" {
			errs["email"] = "Email is required"
		}
		falseVar := false
		views.UserDetails(data, errs, &falseVar).Render(context.Background(), w)
		return
	}

	updatedUser := &db.User{}
	err := db.GetDB().Model(updatedUser).Where("id = ?", user.ID).Updates(map[string]interface{}{"username": username, "email": email}).Error
	if err != nil {
		postgresErr, ok := err.(*pgconn.PgError)
		if !ok {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		formData := views.FormData{"username": username, "email": email}
		wasSuccessful := false

		// Check if the error is a unique constraint violation
		if postgresErr.SQLState() == "23505" {
			constaintArray := strings.Split(postgresErr.ConstraintName, "_")
			columnName := constaintArray[len(constaintArray)-1]

			if columnName == "username" {
				errs := views.FormErrors{"username": "Username is taken"}
				views.UserDetails(formData, errs, &wasSuccessful).Render(context.Background(), w)
				return
			}

			if columnName == "email" {
				errs := views.FormErrors{"email": "Email is taken"}
				views.UserDetails(formData, errs, &wasSuccessful).Render(context.Background(), w)
				return
			}
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	formData := views.FormData{"email": updatedUser.Email, "username": updatedUser.Username}
	truePtr := true
	views.UserDetails(formData, views.FormErrors{}, &truePtr).Render(context.Background(), w)
	return
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

	newOffice := &db.Office{Name: officeName, AdminRefer: user.ID}
	err := db.GetDB().Create(newOffice).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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
	office := &db.Office{}
	err := db.GetDB().Where("code = ?", officeCodeFromPath).
		Preload(clause.Associations).
		Preload("Games.Rankings", func(db *gorm.DB) *gorm.DB {
			return db.Order("Points DESC")
		}).
		Preload("Games.Rankings.User").
		Preload("Games.Matches").
		Preload("Games.Matches.Winner").
		Preload("Games.Matches.Loser").
		First(office).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Office not found", http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user := userFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	selectedGame := office.Games[0]

	userWinLosses := map[uint]views.WinLosses{}
	for _, match := range selectedGame.Matches {
		userWinLosses[match.WinnerID] = views.WinLosses{
			Wins:   userWinLosses[match.WinnerID].Wins + 1,
			Losses: userWinLosses[match.WinnerID].Losses,
		}

		userWinLosses[match.LoserID] = views.WinLosses{
			Wins:   userWinLosses[match.LoserID].Wins,
			Losses: userWinLosses[match.LoserID].Losses + 1,
		}
	}
	officePageContent := views.OfficePage(*office, *user, selectedGame, userWinLosses)
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

	office := &db.Office{}
	err := db.GetDB().Where("code = ?", officeCode).Preload("Players").Preload("Games").First(office).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			data := views.FormData{"office": officeCode}
			errs := views.FormErrors{"office": "Office not found"}
			views.JoinOfficeForm(data, errs).Render(context.Background(), w)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if user is already in the office
	for _, player := range office.Players {
		if player.ID == user.ID {
			data := views.FormData{"office": officeCode}
			errs := views.FormErrors{"office": "You are already in this office"}
			views.JoinOfficeForm(data, errs).Render(context.Background(), w)
			return
		}
	}

	err = db.GetDB().Model(office).Association("Players").Append(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var initRankingsForEachOfficeGame []db.Ranking
	for _, game := range office.Games {
		initRankingsForEachOfficeGame = append(initRankingsForEachOfficeGame, db.Ranking{UserID: user.ID, GameID: game.ID})
	}
	if len(initRankingsForEachOfficeGame) > 0 {
		err = db.GetDB().Model(user).Association("Rankings").Append(initRankingsForEachOfficeGame)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/office/%s", office.Code))
}
