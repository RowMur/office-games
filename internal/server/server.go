package server

import (
	"github.com/labstack/echo/v4"
)

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Run() {
	e := echo.New()

	e.GET("/", authMiddleware(pageHandler))
	e.Static("/static", "assets")

	e.GET("/me", authMiddleware(mePageHandler))
	e.POST("/me", authMiddleware(meUpdateHandler))

	e.GET("/sign-in", enforceSignedOut(signInHandler))
	e.POST("/sign-in", enforceSignedOut(signInFormHandler))

	e.GET("/create-account", enforceSignedOut(createAccountPageHandler))
	e.POST("/create-account", enforceSignedOut(createAccountFormHandler))

	e.GET("/sign-out", signOut)

	e.GET("/offices/:code", authMiddleware(officeHandler))
	e.POST("/offices/join", authMiddleware(joinOfficeHandler))
	e.POST("/offices/create", authMiddleware(createOfficeHandler))

	e.GET("/offices/:code/games/:id", authMiddleware(gamesPageHandler))
	e.GET("/offices/:code/games/create", authMiddleware(enforceAdmin(createGameHandler)))
	e.POST("/offices/:code/games/create", authMiddleware(enforceAdmin(createGameFormHandler)))

	e.GET("/offices/:code/games/:id/play", authMiddleware(gamesPlayPageHandler))
	e.POST("/offices/:code/games/:id/play", authMiddleware(gamesPlayFormHandler))

	e.Logger.Fatal(e.Start(":8080"))
}

// func playHandler(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodPost {
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	user := userFromContext(r.Context())
// 	if user == nil {
// 		http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 		return
// 	}

// 	r.ParseForm()
// 	opponentID, err := strconv.Atoi(r.Form.Get("opponent"))
// 	if err != nil {
// 		http.Error(w, "Invalid opponent", http.StatusBadRequest)
// 		return
// 	}

// 	gameID, err := strconv.Atoi(r.URL.Query().Get("game"))
// 	if err != nil {
// 		http.Error(w, "Invalid game", http.StatusBadRequest)
// 		return
// 	}

// 	opponentIDUint := uint(opponentID)

// 	var winnerID uint
// 	var loserID uint
// 	if r.Form.Get("win") == "on" {
// 		winnerID = user.ID
// 		loserID = opponentIDUint
// 	} else {
// 		winnerID = opponentIDUint
// 		loserID = user.ID
// 	}

// 	match := &db.Match{
// 		GameID:   uint(gameID),
// 		WinnerID: winnerID,
// 		LoserID:  loserID,
// 	}

// 	dbc := db.GetDB()

// 	err = dbc.Create(match).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrForeignKeyViolated) {
// 			http.Error(w, "Invalid game or opponent", http.StatusBadRequest)
// 			return
// 		}

// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	var rankings []db.Ranking
// 	err = dbc.Where("game_id = ?", gameID).Preload("User").Order("Points desc").Find(&rankings).Error
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	game := &db.Game{}
// 	err = dbc.Where("id = ?", gameID).Preload("Matches").First(game).Error
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	userWinLosses := map[uint]views.WinLosses{}
// 	for _, match := range game.Matches {
// 		userWinLosses[match.WinnerID] = views.WinLosses{
// 			Wins:   userWinLosses[match.WinnerID].Wins + 1,
// 			Losses: userWinLosses[match.WinnerID].Losses,
// 		}

// 		userWinLosses[match.LoserID] = views.WinLosses{
// 			Wins:   userWinLosses[match.LoserID].Wins,
// 			Losses: userWinLosses[match.LoserID].Losses + 1,
// 		}
// 	}

// 	views.OfficeRankings(rankings, userWinLosses).Render(context.Background(), w)
// }
