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

	e.Use(authMiddleware)
	signedIn := e.Group("", enforceSignedIn)
	signedOut := e.Group("", enforceSignedOut)
	officeAdmin := e.Group("", enforceAdmin)

	e.GET("/", pageHandler)
	e.Static("/static", "internal/assets")
	e.Static("/", "internal/assets/favicon_io")

	signedIn.GET("/me", mePageHandler)
	signedIn.POST("/me", meUpdateHandler)

	signedOut.GET("/sign-in", signInHandler)
	signedOut.POST("/sign-in", signInFormHandler)

	signedOut.GET("/create-account", createAccountPageHandler)
	signedOut.POST("/create-account", createAccountFormHandler)

	e.GET("/sign-out", signOut)

	signedOut.GET("/forgot-password", forgotPasswordPage)
	signedOut.POST("/forgot-password", forgotPasswordFormHandler)

	signedOut.GET("/reset-password", resetPasswordTokenMiddleware(resetPasswordPage))
	signedOut.POST("/reset-password", resetPasswordTokenMiddleware(resetPasswordFormHandler))

	signedIn.GET("/offices/:code", officeHandler)
	signedIn.POST("/offices/join", joinOfficeHandler)
	signedIn.POST("/offices/create", createOfficeHandler)

	signedIn.GET("/offices/:code/games/:id", gamesPageHandler)
	officeAdmin.GET("/offices/:code/games/create", createGameHandler)
	officeAdmin.POST("/offices/:code/games/create", createGameFormHandler)

	officeAdmin.POST("/offices/:code/games/:id", editGameHandler)
	officeAdmin.DELETE("/offices/:code/games/:id", deleteGameHandler)

	signedIn.GET("/offices/:code/games/:id/play", gamesPlayPageHandler)
	signedIn.POST("/offices/:code/games/:id/play", gamesPlayFormHandler)

	signedIn.GET("/offices/:code/games/:id/pending", gamePendingMatchesPage)
	signedIn.GET("/offices/:code/games/:id/pending/:matchId", pendingMatchPage)
	signedIn.GET("/offices/:code/games/:id/pending/:matchId/approve", pendingMatchApproveHandler)

	officeAdmin.GET("/offices/:code/games/:id/admin", gameAdminPage)

	e.Logger.Fatal(e.Start(":8080"))
}
