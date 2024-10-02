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
	e.Static("/static", "internal/assets")

	e.GET("/me", authMiddleware(mePageHandler))
	e.POST("/me", authMiddleware(meUpdateHandler))

	e.GET("/sign-in", enforceSignedOut(signInHandler))
	e.POST("/sign-in", enforceSignedOut(signInFormHandler))

	e.GET("/create-account", enforceSignedOut(createAccountPageHandler))
	e.POST("/create-account", enforceSignedOut(createAccountFormHandler))

	e.GET("/sign-out", signOut)

	e.GET("/forgot-password", enforceSignedOut(forgotPasswordPage))
	e.POST("/forgot-password", enforceSignedOut(forgotPasswordFormHandler))

	e.GET("/offices/:code", authMiddleware(officeHandler))
	e.POST("/offices/join", authMiddleware(joinOfficeHandler))
	e.POST("/offices/create", authMiddleware(createOfficeHandler))

	e.GET("/offices/:code/games/:id", authMiddleware(gamesPageHandler))
	e.GET("/offices/:code/games/create", authMiddleware(enforceAdmin(createGameHandler)))
	e.POST("/offices/:code/games/create", authMiddleware(enforceAdmin(createGameFormHandler)))

	e.POST("/offices/:code/games/:id", authMiddleware(enforceAdmin(editGameHandler)))
	e.DELETE("/offices/:code/games/:id", authMiddleware(enforceAdmin(deleteGameHandler)))

	e.GET("/offices/:code/games/:id/play", authMiddleware(gamesPlayPageHandler))
	e.POST("/offices/:code/games/:id/play", authMiddleware(gamesPlayFormHandler))

	e.GET("/offices/:code/games/:id/pending", authMiddleware(gamePendingMatchesPage))
	e.GET("/offices/:code/games/:id/pending/:matchId", authMiddleware(pendingMatchPage))
	e.GET("/offices/:code/games/:id/pending/:matchId/approve", authMiddleware(pendingMatchApproveHandler))

	e.GET("/offices/:code/games/:id/admin", authMiddleware(enforceAdmin(gameAdminPage)))

	e.Logger.Fatal(e.Start(":8080"))
}
