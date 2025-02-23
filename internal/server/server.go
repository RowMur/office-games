package server

import (
	"fmt"
	"strings"

	"github.com/RowMur/office-table-tennis/internal/app"
	"github.com/RowMur/office-table-tennis/internal/db"
	"github.com/RowMur/office-table-tennis/internal/officeprocessor"
	"github.com/RowMur/office-table-tennis/internal/user"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	us  *user.UserService
	db  *db.Database
	app *app.App
	op  *officeprocessor.Officeprocessor
}

func NewServer() *Server {
	database := db.Init()
	op := officeprocessor.Newofficeprocessor(database)
	app := app.NewApp(database, op)
	return &Server{
		db:  &database,
		us:  user.NewUserService(database),
		app: app,
		op:  op,
	}
}

func (s *Server) Run() {
	e := echo.New()

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if strings.Contains(c.Request().Host, "office-games") {
				return c.Redirect(301, fmt.Sprintf("https://office-table-tennis.rowmur.dev%s", c.Request().URL.Path))
			}
			return next(c)
		}
	})
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

	officeMember.GET("/offices/:code/play", s.gamesPlayPageHandler)
	officeMember.POST("/offices/:code/play", s.gamesPlayFormHandler)

	officeMember.GET("/offices/:code/pending", s.gamePendingMatchesPage)
	officeMember.GET("/offices/:code/pending/:matchId", s.pendingMatchPage)
	officeMember.GET("/offices/:code/pending/:matchId/approve", s.pendingMatchApproveHandler)
	officeMember.DELETE("/offices/:code/pending/:matchId/delete", s.pendingMatchDeleteHandler)

	officeMember.GET("/offices/:code/matches", s.matchesPageHandler)

	officeMember.GET("/offices/:code/stats", s.gameStatsPageHandler)
	officeMember.POST("/offices/:code/stats", s.gamePlayerStatsPostHandler)

	officeAdmin.POST("/offices/:code/tournaments", s.createTournamentFormHandler)

	e.Any("/offices/:code/games/*", func(c echo.Context) error {
		return c.Redirect(301, "/offices/"+c.Param("code"))
	})

	e.Logger.Fatal(e.Start(":8080"))
}
