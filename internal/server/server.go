package server

import (
	"github.com/RowMur/office-games/internal/app"
	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/gameprocessor"
	"github.com/RowMur/office-games/internal/user"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	us  *user.UserService
	db  *db.Database
	app *app.App
	gp  *gameprocessor.GameProcessor
}

func NewServer() *Server {
	database := db.Init()
	gp := gameprocessor.NewGameProcessor(database)
	app := app.NewApp(database, gp)
	return &Server{
		db:  &database,
		us:  user.NewUserService(database),
		app: app,
		gp:  gp,
	}
}

func (s *Server) Run() {
	e := echo.New()

	e.Use(middleware.CORS())
	e.Use(s.authMiddleware)
	signedIn := e.Group("", enforceSignedIn)
	signedOut := e.Group("", enforceSignedOut)
	officeMember := signedIn.Group("", s.enforceMember)
	officeAdmin := signedIn.Group("", s.enforceAdmin)

	e.GET("/", pageHandler)
	e.GET("/faqs", faqPageHandler)

	e.Static("/static", "internal/assets")
	e.Static("/", "internal/assets/favicon_io")

	signedIn.GET("/me", mePageHandler)
	signedIn.POST("/me", s.meUpdateHandler)

	signedOut.GET("/sign-in", signInHandler)
	signedOut.POST("/sign-in", s.signInFormHandler)

	signedOut.GET("/create-account", createAccountPageHandler)
	signedOut.POST("/create-account", s.createAccountFormHandler)

	e.GET("/sign-out", signOut)

	signedOut.GET("/forgot-password", forgotPasswordPage)
	signedOut.POST("/forgot-password", s.forgotPasswordFormHandler)

	signedOut.GET("/reset-password", resetPasswordTokenMiddleware(resetPasswordPage))
	signedOut.POST("/reset-password", resetPasswordTokenMiddleware(s.resetPasswordFormHandler))

	officeMember.GET("/offices/:code", s.officeHandler)
	signedIn.POST("/offices/join", s.joinOfficeHandler)
	signedIn.POST("/offices/create", s.createOfficeHandler)

	officeMember.GET("/offices/:code/games/:id", s.gamesPageHandler)
	officeAdmin.GET("/offices/:code/games/create", s.createGameHandler)
	officeAdmin.POST("/offices/:code/games/create", s.createGameFormHandler)

	officeAdmin.POST("/offices/:code/games/:id", s.editGameHandler)
	officeAdmin.DELETE("/offices/:code/games/:id", s.deleteGameHandler)

	officeMember.GET("/offices/:code/games/:id/play", s.gamesPlayPageHandler)
	officeMember.POST("/offices/:code/games/:id/play", s.gamesPlayFormHandler)

	officeMember.GET("/offices/:code/games/:id/pending", s.gamePendingMatchesPage)
	officeMember.GET("/offices/:code/games/:id/pending/:matchId", s.pendingMatchPage)
	officeMember.GET("/offices/:code/games/:id/pending/:matchId/approve", s.pendingMatchApproveHandler)
	officeMember.DELETE("/offices/:code/games/:id/pending/:matchId/delete", s.pendingMatchDeleteHandler)

	officeAdmin.GET("/offices/:code/games/:id/admin", s.gameAdminPage)

	officeMember.GET("/offices/:code/games/:id/matches", s.matchesPageHandler)

	officeMember.GET("/offices/:code/games/:id/stats", s.gameStatsPageHandler)
	officeMember.POST("/offices/:code/games/:id/stats", s.gamePlayerStatsPostHandler)

	signedIn.GET("/elo", s.eloPageHandler)

	e.Logger.Fatal(e.Start(":8080"))
}
